package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Protocol IDs for different message types
const (
	ProtocolFileTransfer protocol.ID = "/distribute/file-transfer/1.0.0"
	ProtocolMetadata     protocol.ID = "/distribute/metadata/1.0.0"
	ProtocolReplication  protocol.ID = "/distribute/replication/1.0.0"
)

// Message types for stream communication
const (
	MessageTypeFileRequest  uint32 = 1
	MessageTypeFileResponse uint32 = 2
	MessageTypeChunk        uint32 = 3
	MessageTypeMetadata     uint32 = 4
	MessageTypeReplication  uint32 = 5
)

// StreamHandler handles incoming streams
type StreamHandler func(network.Stream)

// StreamManager manages stream operations
type StreamManager interface {
	// RegisterHandler registers a handler for a protocol
	RegisterHandler(protocol protocol.ID, handler StreamHandler)

	// Send sends a message to a peer
	Send(ctx context.Context, peerID peer.ID, protocol protocol.ID, msg []byte) ([]byte, error)

	// SendStream sends a large payload using streaming
	SendStream(ctx context.Context, peerID peer.ID, protocol protocol.ID, reader io.Reader) error

	// Close shuts down the stream manager
	Close() error
}

// streamManager implements StreamManager
type streamManager struct {
	host     Host
	mu       sync.RWMutex
	handlers map[protocol.ID]StreamHandler
}

// NewStreamManager creates a new stream manager
func NewStreamManager(host Host) StreamManager {
	sm := &streamManager{
		host:     host,
		handlers: make(map[protocol.ID]StreamHandler),
	}

	// Set default handlers
	sm.RegisterHandler(ProtocolFileTransfer, sm.handleFileTransfer)
	sm.RegisterHandler(ProtocolMetadata, sm.handleMetadata)
	sm.RegisterHandler(ProtocolReplication, sm.handleReplication)

	return sm
}

func (sm *streamManager) RegisterHandler(protocol protocol.ID, handler StreamHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[protocol] = handler
	sm.host.SetStreamHandler(protocol, network.StreamHandler(handler))
}

func (sm *streamManager) Send(ctx context.Context, peerID peer.ID, protocol protocol.ID, msg []byte) ([]byte, error) {
	// Open a new stream to the peer
	stream, err := sm.host.NewStream(ctx, peerID, protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	// Write message length prefix
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(msg)))
	if _, err := stream.Write(lengthBuf); err != nil {
		return nil, fmt.Errorf("failed to write message length: %w", err)
	}

	// Write message
	if _, err := stream.Write(msg); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	// Read response length
	if _, err := io.ReadFull(stream, lengthBuf); err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}
	responseLen := binary.BigEndian.Uint32(lengthBuf)

	// Read response
	response := make([]byte, responseLen)
	if _, err := io.ReadFull(stream, response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

func (sm *streamManager) SendStream(ctx context.Context, peerID peer.ID, protocol protocol.ID, reader io.Reader) error {
	// Open a new stream to the peer
	stream, err := sm.host.NewStream(ctx, peerID, protocol)
	if err != nil {
		return fmt.Errorf("failed to open stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	// Copy data to stream
	if _, err := io.Copy(stream, reader); err != nil {
		return fmt.Errorf("failed to copy data to stream: %w", err)
	}

	return nil
}

func (sm *streamManager) Close() error {
	// Remove all handlers
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for protocol := range sm.handlers {
		sm.host.(*libp2pHost).host.RemoveStreamHandler(protocol)
	}

	return nil
}

func (sm *streamManager) handleFileTransfer(stream network.Stream) {
	// Read message length
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(stream, lengthBuf); err != nil {
		fmt.Printf("Failed to read message length: %v\n", err)
		return
	}
	msgLen := binary.BigEndian.Uint32(lengthBuf)

	// Read message
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(stream, msg); err != nil {
		fmt.Printf("Failed to read message: %v\n", err)
		return
	}

	// Process message (simplified)
	fmt.Printf("Received file transfer message from %s\n", stream.Conn().RemotePeer())

	// Send response
	response := []byte("OK")
	lengthBuf = make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(response)))
	if _, err := stream.Write(lengthBuf); err != nil {
		fmt.Printf("Failed to write response length: %v\n", err)
		return
	}
	if _, err := stream.Write(response); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
		return
	}
}

func (sm *streamManager) handleMetadata(stream network.Stream) {
	// Read message length
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(stream, lengthBuf); err != nil {
		fmt.Printf("Failed to read message length: %v\n", err)
		return
	}
	msgLen := binary.BigEndian.Uint32(lengthBuf)

	// Read message
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(stream, msg); err != nil {
		fmt.Printf("Failed to read message: %v\n", err)
		return
	}

	// Process message (simplified)
	fmt.Printf("Received metadata message from %s\n", stream.Conn().RemotePeer())

	// Send response
	response := []byte("OK")
	lengthBuf = make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(response)))
	if _, err := stream.Write(lengthBuf); err != nil {
		fmt.Printf("Failed to write response length: %v\n", err)
		return
	}
	if _, err := stream.Write(response); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
		return
	}
}

func (sm *streamManager) handleReplication(stream network.Stream) {
	// Read message length
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(stream, lengthBuf); err != nil {
		fmt.Printf("Failed to read message length: %v\n", err)
		return
	}
	msgLen := binary.BigEndian.Uint32(lengthBuf)

	// Read message
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(stream, msg); err != nil {
		fmt.Printf("Failed to read message: %v\n", err)
		return
	}

	// Process message (simplified)
	fmt.Printf("Received replication message from %s\n", stream.Conn().RemotePeer())

	// Send response
	response := []byte("OK")
	lengthBuf = make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(response)))
	if _, err := stream.Write(lengthBuf); err != nil {
		fmt.Printf("Failed to write response length: %v\n", err)
		return
	}
	if _, err := stream.Write(response); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
		return
	}
}
