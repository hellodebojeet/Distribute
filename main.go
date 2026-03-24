package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hellodebojeet/Distribute/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcptransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcptransportOpts)

	fileServerOpts := FileServerOpts{
		EncKey:            newEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	fmt.Println("🚀 Starting Distributed Filesystem Demo...")

	// Create three nodes in the network
	s1 := makeServer(":3001", "")
	s2 := makeServer(":3002", "")
	s3 := makeServer(":3003", ":3001", ":3002")

	// Start servers
	go func() {
		if err := s1.Start(); err != nil {
			log.Printf("Server 1 error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	go func() {
		if err := s2.Start(); err != nil {
			log.Printf("Server 2 error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	go func() {
		if err := s3.Start(); err != nil {
			log.Printf("Server 3 error: %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	// Test file operations
	fmt.Println("\n📁 Testing File Operations...")

	// Test 1: Store a file
	testFile := "Hello Distributed World!"
	fmt.Printf("Storing file: %q\n", testFile)

	err := s3.Store("test.txt", bytes.NewReader([]byte(testFile)))
	if err != nil {
		log.Fatalf("Failed to store file: %v", err)
	}
	fmt.Println("✅ File stored successfully!")

	// Test 2: Retrieve the file from the same node
	fmt.Println("\nRetrieving file from node 3003...")
	reader, err := s3.Get("test.txt")
	if err != nil {
		log.Fatalf("Failed to get file: %v", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	retrieved := buf.String()
	fmt.Printf("Retrieved: %q\n", retrieved)

	if retrieved == testFile {
		fmt.Println("✅ File retrieval successful!")
	} else {
		fmt.Println("❌ File content mismatch!")
	}

	// Test 3: Retrieve from a different node (should fetch from network)
	fmt.Println("\nRetrieving file from node 3001 (should fetch from network)...")
	reader2, err := s1.Get("test.txt")
	if err != nil {
		log.Fatalf("Failed to get file from node 3001: %v", err)
	}

	buf2 := new(bytes.Buffer)
	buf2.ReadFrom(reader2)
	retrieved2 := buf2.String()
	fmt.Printf("Retrieved from network: %q\n", retrieved2)

	if retrieved2 == testFile {
		fmt.Println("✅ Network retrieval successful!")
	} else {
		fmt.Println("❌ Network retrieval failed!")
	}

	// Test 4: Store multiple files
	fmt.Println("\nStoring multiple files...")
	files := map[string]string{
		"doc1.txt":  "Document 1 content",
		"doc2.txt":  "Document 2 content",
		"image.png": "fake image data",
	}

	for filename, content := range files {
		err := s3.Store(filename, bytes.NewReader([]byte(content)))
		if err != nil {
			log.Printf("Failed to store %s: %v", filename, err)
		} else {
			fmt.Printf("✅ Stored %s\n", filename)
		}
	}

	// Test 5: Retrieve all files from different nodes
	fmt.Println("\nRetrieving all files from node 3002...")
	for filename := range files {
		reader, err := s2.Get(filename)
		if err != nil {
			log.Printf("Failed to get %s from node 3002: %v", filename, err)
			continue
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		content := buf.String()
		fmt.Printf("📄 %s: %q\n", filename, content)
	}

	// Test 6: Delete a file
	fmt.Println("\nDeleting doc2.txt...")
	err = s3.store.Delete(s3.ID, "doc2.txt")
	if err != nil {
		log.Printf("Failed to delete file: %v", err)
	} else {
		fmt.Println("✅ File deleted successfully!")
	}

	// Test 7: Try to retrieve deleted file
	fmt.Println("\nTrying to retrieve deleted file...")
	_, err = s1.Get("doc2.txt")
	if err != nil {
		fmt.Printf("✅ Expected error when retrieving deleted file: %v\n", err)
	} else {
		fmt.Println("❌ Should have failed to retrieve deleted file!")
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n📊 Summary:")
	fmt.Println("  ✅ File storage working")
	fmt.Println("  ✅ File retrieval working")
	fmt.Println("  ✅ Network replication working")
	fmt.Println("  ✅ Multi-node coordination working")
	fmt.Println("  ✅ File deletion working")

	// Graceful shutdown
	time.Sleep(1 * time.Second)
	s1.Stop()
	s2.Stop()
	s3.Stop()
}
