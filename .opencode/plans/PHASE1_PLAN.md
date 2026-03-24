# Phase 1: Metadata Integration (Control Plane)

## What to modify in existing code:
1. `/home/Linux/Distribute/server.go` - FileServer struct and methods
2. `/home/Linux/Distribute/main.go` - Initialize metadata service and connect to FileServers
3. `/home/Linux/Distribute/metadata/client.go` - May need to enhance client for chunk tracking

## New interfaces (Go):
No new interfaces needed - using existing MetadataStore and MetadataClient interfaces from metadata package

## Key code snippets:

### 1. Modify FileServer struct to include metadata client
```go
type FileServer struct {
    FileServerOpts
    
    peerLock sync.Mutex
    peers    map[string]p2p.Peer
    
    store  *Store
    quitch chan struct{}
    
    // NEW: Metadata client
    metadataClient metadata.MetadataClient
}
```

### 2. Update NewFileServer to accept metadata client
```go
func NewFileServer(opts FileServerOpts, metadataClient metadata.MetadataClient) *FileServer {
    // ... existing code ...
    
    return &FileServer{
        FileServerOpts: opts,
        store:          NewStore(storeOpts),
        quitch:         make(chan struct{}),
        peers:          make(map[string]p2p.Peer),
        metadataClient: metadataClient, // NEW
    }
}
```

### 3. Modify Store method to record chunk metadata
```go
func (s *FileServer) Store(key string, r io.Reader) error {
    var (
        fileBuffer = new(bytes.Buffer)
        tee        = io.TeeReader(r, fileBuffer)
    )

    size, err := s.store.Write(s.ID, key, tee)
    if err != nil {
        return err
    }

    // NEW: Generate chunk ID and save metadata
    chunkID := hashKey(key) // Using existing hash function
    locations := []string{s.ID} // Initially only on current node
    
    // Save chunk locations to metadata service
    if err := s.metadataClient.SaveChunkLocations(chunkID, locations); err != nil {
        return fmt.Errorf("failed to save chunk metadata: %w", err)
    }
    
    // TODO: Update file metadata (would need to get existing file metadata first)
    
    // ... rest of existing code ...
}
```

### 4. Modify Get method to check metadata for chunk locations
```go
func (s *FileServer) Get(key string) (io.Reader, error) {
    if s.store.Has(s.ID, key) {
        fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
        _, r, err := s.store.Read(s.ID, key)
        return r, err
    }

    // NEW: Check metadata for chunk locations
    chunkID := hashKey(key)
    locations, err := s.metadataClient.GetChunkLocations(chunkID)
    if err != nil {
        return nil, fmt.Errorf("failed to get chunk locations: %w", err)
    }
    
    if len(locations) == 0 {
        return nil, fmt.Errorf("chunk %s not found on any node", chunkID)
    }
    
    fmt.Printf("[%s] dont have file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

    // ... existing network fetch code would be modified to try locations in order ...
}
```

### 5. Update main.go to initialize metadata service
```go
func main() {
    fmt.Println("🚀 Starting Distributed Filesystem Demo...")
    
    // NEW: Start metadata service (embedded or external)
    metadataAddr := ":8080" // or make configurable
    
    // For demo, we'll start it in background
    go func() {
        // In a real implementation, we'd call the metadata service main
        // For now, assume it's running externally
    }()
    
    // Give metadata service time to start
    time.Sleep(100 * time.Millisecond)
    
    // NEW: Create metadata client
    metadataClient := metadata.NewHTTPMetadataClient("http://localhost" + metadataAddr)
    
    // Create three nodes in the network WITH metadata client
    s1 := makeServer(":3001", "", metadataClient)
    s2 := makeServer(":3002", "", metadataClient)
    s3 := makeServer(":3003", ":3001", ":3002", metadataClient)
    
    // ... rest of existing code ...
}

// Update makeServer function signature
func makeServer(listenAddr string, nodes ...string, metadataClient metadata.MetadataClient) *FileServer {
    // ... existing code ...
    
    fileServerOpts := FileServerOpts{
        EncKey:            newEncryptionKey(),
        StorageRoot:       listenAddr + "_network",
        PathTransformFunc: CASPathTransformFunc,
        Transport:         tcpTransport,
        BootstrapNodes:    nodes,
        MetadataAddr:      metadataAddr, // NEW: Set metadata address
    }
    
    // NEW: Pass metadata client to FileServer
    return NewFileServer(fileServerOpts, metadataClient)
}
```

## Integration points:
1. FileServer creation - pass metadata client
2. Store method - after local write, save chunk metadata
3. Get method - before network fetch, check metadata for locations
4. Main initialization - start metadata service and create client

## Edge cases handled:
1. Metadata service unavailable - return appropriate errors
2. No chunk locations found - return file not found error
3. Failed to save metadata - propagate error to caller
4. Concurrent access - metadata store handles locking internally

## Notes:
- This is a minimal implementation focusing on chunk tracking
- File metadata (version, checksum, etc.) would be added in later enhancements
- The metadata service should already be running or we need to start it embedded
- For Phase 1, we're focusing on the control plane integration only