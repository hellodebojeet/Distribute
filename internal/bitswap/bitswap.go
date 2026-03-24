package bitswap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hellodebojeet/Distribute/internal/blockstore"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Bitswap represents a block exchange protocol
type Bitswap interface {
	// GetBlock retrieves a block from the network
	GetBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error)

	// HasBlock announces that we have a block
	HasBlock(c cid.Cid)

	// WantBlock requests a block from the network
	WantBlock(c cid.Cid)

	// Close shuts down the bitswap service
	Close() error
}

// Exchange provides a high-level interface for block exchange
type Exchange interface {
	// FetchBlock fetches a block from the network
	FetchBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error)

	// ProvideBlock announces that we can provide a block
	ProvideBlock(ctx context.Context, b *blockstore.Block) error

	// RegisterProvider registers a provider for a block
	RegisterProvider(c cid.Cid, provider peer.ID)

	// GetProviders gets providers for a block
	GetProviders(c cid.Cid) []peer.ID
}

// bitswapImpl implements Bitswap
type bitswapImpl struct {
	host       host.Host
	blockstore blockstore.BlockStore
	cache      blockstore.Cache
	mu         sync.RWMutex
	wantList   map[cid.Cid]struct{}
	haveList   map[cid.Cid]struct{}
	providers  map[cid.Cid]map[peer.ID]struct{}
}

// BitswapConfig holds configuration for bitswap
type BitswapConfig struct {
	Host       host.Host
	Blockstore blockstore.BlockStore
	CacheSize  int
}

// NewBitswap creates a new bitswap instance
func NewBitswap(cfg BitswapConfig) (Bitswap, error) {
	if cfg.Host == nil {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.Blockstore == nil {
		return nil, fmt.Errorf("blockstore is required")
	}

	cacheSize := cfg.CacheSize
	if cacheSize <= 0 {
		cacheSize = 1000
	}

	return &bitswapImpl{
		host:       cfg.Host,
		blockstore: cfg.Blockstore,
		cache:      blockstore.NewCache(cacheSize),
		wantList:   make(map[cid.Cid]struct{}),
		haveList:   make(map[cid.Cid]struct{}),
		providers:  make(map[cid.Cid]map[peer.ID]struct{}),
	}, nil
}

func (bs *bitswapImpl) GetBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error) {
	// Check cache first
	if block, exists := bs.cache.Get(c); exists {
		return block, nil
	}

	// Check local blockstore
	if exists, err := bs.blockstore.Has(ctx, c); err == nil && exists {
		block, err := bs.blockstore.Get(ctx, c)
		if err != nil {
			return nil, err
		}
		// Add to cache
		bs.cache.Put(block)
		return block, nil
	}

	// Request from network
	bs.mu.Lock()
	bs.wantList[c] = struct{}{}
	bs.mu.Unlock()

	// Wait for block (simplified)
	// In practice, you would implement proper network communication
	// For now, we'll simulate a timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for block: %s", c)
	}
}

func (bs *bitswapImpl) HasBlock(c cid.Cid) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.haveList[c] = struct{}{}
	delete(bs.wantList, c)
}

func (bs *bitswapImpl) WantBlock(c cid.Cid) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.wantList[c] = struct{}{}
}

func (bs *bitswapImpl) Close() error {
	return nil
}

// exchangeImpl implements Exchange
type exchangeImpl struct {
	bitswap    Bitswap
	blockstore blockstore.BlockStore
	mu         sync.RWMutex
	providers  map[cid.Cid]map[peer.ID]struct{}
}

// NewExchange creates a new exchange instance
func NewExchange(bs Bitswap, blockstore blockstore.BlockStore) Exchange {
	return &exchangeImpl{
		bitswap:    bs,
		blockstore: blockstore,
		providers:  make(map[cid.Cid]map[peer.ID]struct{}),
	}
}

func (e *exchangeImpl) FetchBlock(ctx context.Context, c cid.Cid) (*blockstore.Block, error) {
	return e.bitswap.GetBlock(ctx, c)
}

func (e *exchangeImpl) ProvideBlock(ctx context.Context, b *blockstore.Block) error {
	// Store block locally
	if err := e.blockstore.Put(ctx, b); err != nil {
		return err
	}

	// Announce that we have the block
	e.bitswap.HasBlock(b.Cid)

	return nil
}

func (e *exchangeImpl) RegisterProvider(c cid.Cid, provider peer.ID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.providers[c] == nil {
		e.providers[c] = make(map[peer.ID]struct{})
	}
	e.providers[c][provider] = struct{}{}
}

func (e *exchangeImpl) GetProviders(c cid.Cid) []peer.ID {
	e.mu.RLock()
	defer e.mu.RUnlock()

	providers := make([]peer.ID, 0, len(e.providers[c]))
	for p := range e.providers[c] {
		providers = append(providers, p)
	}
	return providers
}

// WantManager manages wanted blocks
type WantManager interface {
	// Want adds a block to the want list
	Want(c cid.Cid)

	// CancelWant removes a block from the want list
	CancelWant(c cid.Cid)

	// GetWants returns the current want list
	GetWants() []cid.Cid

	// IsWanted checks if a block is wanted
	IsWanted(c cid.Cid) bool
}

// wantManager implements WantManager
type wantManager struct {
	mu    sync.RWMutex
	wants map[cid.Cid]struct{}
}

// NewWantManager creates a new want manager
func NewWantManager() WantManager {
	return &wantManager{
		wants: make(map[cid.Cid]struct{}),
	}
}

func (wm *wantManager) Want(c cid.Cid) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.wants[c] = struct{}{}
}

func (wm *wantManager) CancelWant(c cid.Cid) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	delete(wm.wants, c)
}

func (wm *wantManager) GetWants() []cid.Cid {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wants := make([]cid.Cid, 0, len(wm.wants))
	for c := range wm.wants {
		wants = append(wants, c)
	}
	return wants
}

func (wm *wantManager) IsWanted(c cid.Cid) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	_, exists := wm.wants[c]
	return exists
}
