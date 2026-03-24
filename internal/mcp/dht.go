package mcp

import (
	"context"

	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHT represents a Kademlia DHT interface for peer discovery and routing.
type DHT interface {
	// Bootstrap the DHT with known peers.
	Bootstrap(context.Context) error

	// FindPeersAsync searches for peers in the DHT.
	FindPeersAsync(ctx context.Context, peerID peer.ID) <-chan peer.AddrInfo

	// FindProvidersAsync searches for providers of a key.
	FindProvidersAsync(ctx context.Context, key cid.Cid, maxProviders int) <-chan peer.AddrInfo

	// Provide announces that we are providing a key.
	Provide(ctx context.Context, key cid.Cid, announce bool) error

	// Close shuts down the DHT.
	Close() error
}

// kadDHT is the implementation of DHT using libp2p-kad-dht.
type kadDHT struct {
	*kaddht.IpfsDHT
}

// NewKadDHT creates a new KadDHT instance.
func NewKadDHT(host host.Host) (DHT, error) {
	// Create DHT options
	opts := []kaddht.Option{
		kaddht.Mode(kaddht.ModeAuto), // Auto client/server mode
		kaddht.ProtocolPrefix("/ipfs/kad/1.0.0"),
	}

	// Create the DHT
	dhtInstance, err := kaddht.New(context.Background(), host, opts...)
	if err != nil {
		return nil, err
	}

	return &kadDHT{IpfsDHT: dhtInstance}, nil
}

// Bootstrap the DHT with known peers.
func (d *kadDHT) Bootstrap(ctx context.Context) error {
	return d.IpfsDHT.Bootstrap(ctx)
}

// FindPeersAsync searches for peers in the DHT.
func (d *kadDHT) FindPeersAsync(ctx context.Context, peerID peer.ID) <-chan peer.AddrInfo {
	// FindPeer returns (peer.AddrInfo, error), so we need to convert it to a channel
	result := make(chan peer.AddrInfo)
	go func() {
		defer close(result)
		if peerInfo, err := d.IpfsDHT.FindPeer(ctx, peerID); err == nil {
			result <- peerInfo
		}
	}()
	return result
}

// FindProvidersAsync searches for providers of a key.
func (d *kadDHT) FindProvidersAsync(ctx context.Context, key cid.Cid, maxProviders int) <-chan peer.AddrInfo {
	return d.IpfsDHT.FindProvidersAsync(ctx, key, maxProviders)
}

// Provide announces that we are providing a key.
func (d *kadDHT) Provide(ctx context.Context, key cid.Cid, announce bool) error {
	return d.IpfsDHT.Provide(ctx, key, announce)
}

// Close shuts down the DHT.
func (d *kadDHT) Close() error {
	return d.IpfsDHT.Close()
}
