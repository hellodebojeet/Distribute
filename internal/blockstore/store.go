package blockstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

// Block represents a block of data with its CID
type Block struct {
	Cid  cid.Cid
	Data []byte
}

// BlockStore represents a block storage interface
type BlockStore interface {
	// Has checks if a block exists in the store
	Has(ctx context.Context, c cid.Cid) (bool, error)

	// Get retrieves a block from the store
	Get(ctx context.Context, c cid.Cid) (*Block, error)

	// Put stores a block in the store
	Put(ctx context.Context, b *Block) error

	// Delete removes a block from the store
	Delete(ctx context.Context, c cid.Cid) error

	// Close closes the block store
	Close() error
}

// blockStoreImpl implements BlockStore
type blockStoreImpl struct {
	ds datastore.Datastore
	mu sync.RWMutex
}

// BlockStoreConfig holds configuration for the block store
type BlockStoreConfig struct {
	Datastore datastore.Datastore
}

// NewBlockStore creates a new block store
func NewBlockStore(cfg BlockStoreConfig) (BlockStore, error) {
	if cfg.Datastore == nil {
		return nil, fmt.Errorf("datastore is required")
	}

	return &blockStoreImpl{
		ds: cfg.Datastore,
	}, nil
}

func (bs *blockStoreImpl) Has(ctx context.Context, c cid.Cid) (bool, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	key := datastore.NewKey(c.String())
	return bs.ds.Has(ctx, key)
}

func (bs *blockStoreImpl) Get(ctx context.Context, c cid.Cid) (*Block, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	key := datastore.NewKey(c.String())
	data, err := bs.ds.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return &Block{
		Cid:  c,
		Data: data,
	}, nil
}

func (bs *blockStoreImpl) Put(ctx context.Context, b *Block) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := datastore.NewKey(b.Cid.String())
	return bs.ds.Put(ctx, key, b.Data)
}

func (bs *blockStoreImpl) Delete(ctx context.Context, c cid.Cid) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := datastore.NewKey(c.String())
	return bs.ds.Delete(ctx, key)
}

func (bs *blockStoreImpl) Close() error {
	return bs.ds.Close()
}

// Cache provides caching functionality for blocks
type Cache interface {
	// Get retrieves a block from the cache
	Get(c cid.Cid) (*Block, bool)

	// Put adds a block to the cache
	Put(b *Block)

	// Remove removes a block from the cache
	Remove(c cid.Cid)

	// Clear clears the cache
	Clear()

	// Size returns the current size of the cache
	Size() int
}

// simpleCache implements Cache using a simple map
type simpleCache struct {
	mu     sync.RWMutex
	blocks map[string]*Block
	size   int
}

// NewCache creates a new cache with the specified size
func NewCache(size int) Cache {
	return &simpleCache{
		blocks: make(map[string]*Block),
		size:   size,
	}
}

func (c *simpleCache) Get(cid cid.Cid) (*Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	block, exists := c.blocks[cid.String()]
	return block, exists
}

func (c *simpleCache) Put(b *Block) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If cache is full, remove oldest (simplified)
	if len(c.blocks) >= c.size {
		// Remove first item (simplified LRU)
		for k := range c.blocks {
			delete(c.blocks, k)
			break
		}
	}

	c.blocks[b.Cid.String()] = b
}

func (c *simpleCache) Remove(cid cid.Cid) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.blocks, cid.String())
}

func (c *simpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blocks = make(map[string]*Block)
}

func (c *simpleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.blocks)
}
