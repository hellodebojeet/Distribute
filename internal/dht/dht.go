package dht

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	routed "github.com/libp2p/go-libp2p/p2p/host/routed"
)

// DHT represents a Kademlia Distributed Hash Table interface
type DHT interface {
	// Bootstrap starts the DHT bootstrap process
	Bootstrap(ctx context.Context) error

	// PutValue stores a key-value pair in the DHT
	PutValue(ctx context.Context, key string, value []byte) error

	// GetValue retrieves a value by key from the DHT
	GetValue(ctx context.Context, key string) ([]byte, error)

	// FindPeer finds a specific peer by ID
	FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error)

	// RoutingTable returns the DHT routing table
	RoutingTable() interface{}

	// Close shuts down the DHT
	Close() error
}

// kadDHTWrapper wraps a libp2p Kademlia DHT
type kadDHTWrapper struct {
	dht  *dht.IpfsDHT
	host host.Host
}

// DHTConfig holds configuration for the DHT
type DHTConfig struct {
	Host           host.Host
	Mode           dht.ModeOpt
	BootstrapPeers []peer.AddrInfo
	BucketSize     int
	Concurrency    int
	Timeout        time.Duration
}

// NewDHT creates a new Kademlia DHT instance
func NewDHT(cfg DHTConfig) (DHT, error) {
	var opts []dht.Option

	// Set mode
	if cfg.Mode != 0 {
		opts = append(opts, dht.Mode(cfg.Mode))
	} else {
		opts = append(opts, dht.Mode(dht.ModeAutoServer))
	}

	// Set bucket size
	if cfg.BucketSize > 0 {
		opts = append(opts, dht.BucketSize(cfg.BucketSize))
	}

	// Set concurrency
	if cfg.Concurrency > 0 {
		opts = append(opts, dht.Concurrency(cfg.Concurrency))
	}

	// Create the DHT
	kadDHT, err := dht.New(context.Background(), cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT
	if err := kadDHT.Bootstrap(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	return &kadDHTWrapper{
		dht:  kadDHT,
		host: cfg.Host,
	}, nil
}

// NewRoutedDHT creates a new routed host with DHT
func NewRoutedDHT(h host.Host, cfg DHTConfig) (host.Host, DHT, error) {
	// Create DHT
	d, err := NewDHT(cfg)
	if err != nil {
		return nil, nil, err
	}

	// Create routed host
	dWrapper := d.(*kadDHTWrapper)
	routedHost := routed.Wrap(h, dWrapper.dht)

	return routedHost, d, nil
}

func (d *kadDHTWrapper) Bootstrap(ctx context.Context) error {
	return d.dht.Bootstrap(ctx)
}

func (d *kadDHTWrapper) PutValue(ctx context.Context, key string, value []byte) error {
	return d.dht.PutValue(ctx, key, value)
}

func (d *kadDHTWrapper) GetValue(ctx context.Context, key string) ([]byte, error) {
	return d.dht.GetValue(ctx, key)
}

func (d *kadDHTWrapper) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	return d.dht.FindPeer(ctx, peerID)
}

func (d *kadDHTWrapper) RoutingTable() interface{} {
	return d.dht.RoutingTable()
}

func (d *kadDHTWrapper) Close() error {
	return d.dht.Close()
}
