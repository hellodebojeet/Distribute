package mcp

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

// P2PNode represents a libp2p node interface for the distributed filesystem.
type P2PNode interface {
	// Context returns the node's context.
	Context() context.Context

	// ID returns the node's peer ID.
	ID() peer.ID

	// Addrs returns the node's listening addresses.
	Addrs() []peer.AddrInfo

	// Connect establishes a connection to a peer.
	Connect(ctx context.Context, p peer.AddrInfo) error

	// Disconnect closes connection to a peer.
	Disconnect(p peer.ID) error

	// SetStreamHandler sets a handler for a specific protocol.
	SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler)

	// RemoveStreamHandler removes a handler for a specific protocol.
	RemoveStreamHandler(protocolID protocol.ID)

	// NewStream opens a new stream to a peer for a protocol.
	NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)

	// GetHost returns the underlying libp2p host.
	GetHost() host.Host

	// Close shuts down the node.
	Close() error
}

// Libp2pNode is the implementation of P2PNode using libp2p.
type Libp2pNode struct {
	host host.Host
}

// NewLibp2pNode creates a new libp2p node with the given listen port and private key.
func NewLibp2pNode(listenPort int, privKey []byte) (P2PNode, error) {
	// Convert private key bytes to libp2p private key
	prvKey, err := crypto.UnmarshalPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	// Create libp2p options
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(prvKey),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	}

	// Create the host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Libp2pNode{host: h}, nil
}

// Context returns the node's context.
func (n *Libp2pNode) Context() context.Context {
	return context.Background() // Simplified for now
}

// ID returns the node's peer ID.
func (n *Libp2pNode) ID() peer.ID {
	return n.host.ID()
}

// Addrs returns the node's listening addresses.
func (n *Libp2pNode) Addrs() []peer.AddrInfo {
	addrs := make([]peer.AddrInfo, 0)
	for _, addr := range n.host.Addrs() {
		addrs = append(addrs, peer.AddrInfo{ID: n.host.ID(), Addrs: []ma.Multiaddr{addr}})
	}
	return addrs
}

// Connect establishes a connection to a peer.
func (n *Libp2pNode) Connect(ctx context.Context, p peer.AddrInfo) error {
	return n.host.Connect(ctx, p)
}

// Disconnect closes connection to a peer.
func (n *Libp2pNode) Disconnect(p peer.ID) error {
	n.host.Network().ClosePeer(p)
	return nil
}

// SetStreamHandler sets a handler for a specific protocol.
func (n *Libp2pNode) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	n.host.SetStreamHandler(protocolID, handler)
}

// RemoveStreamHandler removes a handler for a specific protocol.
func (n *Libp2pNode) RemoveStreamHandler(protocolID protocol.ID) {
	n.host.RemoveStreamHandler(protocolID)
}

// NewStream opens a new stream to a peer for a protocol.
func (n *Libp2pNode) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	return n.host.NewStream(ctx, p, pids...)
}

// GetHost returns the underlying libp2p host.
func (n *Libp2pNode) GetHost() host.Host {
	return n.host
}

// Close shuts down the node.
func (n *Libp2pNode) Close() error {
	return n.host.Close()
}
