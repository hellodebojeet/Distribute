package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Discovery handles peer discovery mechanisms
type Discovery interface {
	// Bootstrap starts the discovery process with bootstrap nodes
	Bootstrap(ctx context.Context, bootstrapPeers []peer.AddrInfo) error

	// Advertise advertises this node for a given service
	Advertise(ctx context.Context, ns string) error

	// FindPeers finds peers providing a given service
	FindPeers(ctx context.Context, ns string) (<-chan peer.AddrInfo, error)

	// RegisterNotifee registers a notifee for peer discovery events
	RegisterNotifee(notifee DiscoveryNotifee)

	// Close stops the discovery service
	Close() error
}

// DiscoveryNotifee receives notifications about discovered peers
type DiscoveryNotifee interface {
	// HandlePeerFound is called when a new peer is discovered
	HandlePeerFound(peer.AddrInfo)
}

// discoveryService implements Discovery
type discoveryService struct {
	host     Host
	mu       sync.RWMutex
	notifees []DiscoveryNotifee
	cancel   context.CancelFunc
}

// DiscoveryConfig holds configuration for discovery
type DiscoveryConfig struct {
	Host            Host
	BootstrapPeers  []string
	Rendezvous      string
	DiscoveryPeriod time.Duration
}

// NewDiscovery creates a new discovery service
func NewDiscovery(cfg DiscoveryConfig) (Discovery, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ds := &discoveryService{
		host:   cfg.Host,
		cancel: cancel,
	}

	// Parse bootstrap peers if provided
	var bootstrapPeers []peer.AddrInfo
	if len(cfg.BootstrapPeers) > 0 {
		for _, addr := range cfg.BootstrapPeers {
			maddr, err := ma.NewMultiaddr(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid bootstrap address %s: %w", addr, err)
			}
			pi, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bootstrap address %s: %w", addr, err)
			}
			bootstrapPeers = append(bootstrapPeers, *pi)
		}
	}

	// Start bootstrap in background
	go func() {
		if err := ds.Bootstrap(ctx, bootstrapPeers); err != nil {
			fmt.Printf("Bootstrap error: %v\n", err)
		}
	}()

	return ds, nil
}

func (ds *discoveryService) Bootstrap(ctx context.Context, bootstrapPeers []peer.AddrInfo) error {
	if len(bootstrapPeers) == 0 {
		return nil
	}

	h := ds.host.(*libp2pHost).host

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	for _, pi := range bootstrapPeers {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			if err := h.Connect(ctx, pi); err != nil {
				fmt.Printf("Failed to connect to bootstrap peer %s: %v\n", pi.ID, err)
			} else {
				fmt.Printf("Connected to bootstrap peer %s\n", pi.ID)
			}
		}(pi)
	}
	wg.Wait()

	return nil
}

func (ds *discoveryService) Advertise(ctx context.Context, ns string) error {
	// TODO: Implement advertise using libp2p DHT or mDNS
	fmt.Printf("Advertising for namespace: %s\n", ns)
	return nil
}

func (ds *discoveryService) FindPeers(ctx context.Context, ns string) (<-chan peer.AddrInfo, error) {
	// TODO: Implement peer discovery using libp2p DHT or mDNS
	ch := make(chan peer.AddrInfo)
	close(ch)
	return ch, nil
}

func (ds *discoveryService) RegisterNotifee(notifee DiscoveryNotifee) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.notifees = append(ds.notifees, notifee)
}

func (ds *discoveryService) Close() error {
	ds.cancel()
	return nil
}
