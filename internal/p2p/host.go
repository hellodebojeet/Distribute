package p2p

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

// Host represents a libp2p node in the network
type Host interface {
	// ID returns the peer ID of this node
	ID() peer.ID

	// Addrs returns the listening addresses of this node
	Addrs() []ma.Multiaddr

	// Connect connects to a peer given its address info
	Connect(ctx context.Context, pi peer.AddrInfo) error

	// Disconnect disconnects from a peer
	Disconnect(pid peer.ID) error

	// SetStreamHandler sets a handler for a given protocol
	SetStreamHandler(pid protocol.ID, handler network.StreamHandler)

	// NewStream opens a new stream to a peer for a given protocol
	NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)

	// Close shuts down the host
	Close() error

	// Peerstore returns the peerstore
	Peerstore() peerstore.Peerstore

	// Mux returns the stream multiplexer
	Mux() protocol.Switch

	// Network returns the network interface
	Network() network.Network
}

// libp2pHost wraps a libp2p host
type libp2pHost struct {
	host host.Host
}

// HostConfig holds configuration for creating a new host
type HostConfig struct {
	ListenAddrs []string
	PrivKey     crypto.PrivKey
	NATManager  bool
	Relay       bool
}

// NewHost creates a new libp2p host
func NewHost(cfg HostConfig) (Host, error) {
	var opts []libp2p.Option

	// Set listen addresses
	if len(cfg.ListenAddrs) > 0 {
		addrs := make([]ma.Multiaddr, 0, len(cfg.ListenAddrs))
		for _, addr := range cfg.ListenAddrs {
			maddr, err := ma.NewMultiaddr(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid listen address %s: %w", addr, err)
			}
			addrs = append(addrs, maddr)
		}
		opts = append(opts, libp2p.ListenAddrs(addrs...))
	}

	// Set identity
	if cfg.PrivKey != nil {
		opts = append(opts, libp2p.Identity(cfg.PrivKey))
	}

	// Enable NAT management
	if cfg.NATManager {
		opts = append(opts, libp2p.EnableNATService())
	}

	// Enable relay
	if cfg.Relay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// Use default transports, muxers, and security
	opts = append(opts,
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	)

	// Create the host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	return &libp2pHost{host: h}, nil
}

// NewHostWithKey creates a new libp2p host with a generated key
func NewHostWithKey(listenAddr string) (Host, crypto.PrivKey, error) {
	// Generate a new key pair
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	cfg := HostConfig{
		ListenAddrs: []string{listenAddr},
		PrivKey:     priv,
		NATManager:  true,
		Relay:       true,
	}

	h, err := NewHost(cfg)
	if err != nil {
		return nil, nil, err
	}

	return h, priv, nil
}

func (h *libp2pHost) ID() peer.ID {
	return h.host.ID()
}

func (h *libp2pHost) Addrs() []ma.Multiaddr {
	return h.host.Addrs()
}

func (h *libp2pHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return h.host.Connect(ctx, pi)
}

func (h *libp2pHost) Disconnect(pid peer.ID) error {
	return h.host.Network().ClosePeer(pid)
}

func (h *libp2pHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	h.host.SetStreamHandler(pid, handler)
}

func (h *libp2pHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	return h.host.NewStream(ctx, p, pids...)
}

func (h *libp2pHost) Close() error {
	return h.host.Close()
}

func (h *libp2pHost) Peerstore() peerstore.Peerstore {
	return h.host.Peerstore()
}

func (h *libp2pHost) Mux() protocol.Switch {
	return h.host.Mux()
}

func (h *libp2pHost) Network() network.Network {
	return h.host.Network()
}
