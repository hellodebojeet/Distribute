// Package blockstore provides a block storage interface for CID-addressed data.
package blockstore

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/hellodebojeet/Distribute/server"
	"github.com/ipfs/go-cid"
)

// BlockStore provides CID-addressed block storage backed by the existing Store.
type BlockStore struct {
	store  *server.Store
	mu     sync.RWMutex
	pinned map[cid.Cid]bool
}

// NewBlockStore creates a new block store.
func NewBlockStore(store *server.Store) *BlockStore {
	return &BlockStore{
		store:  store,
		pinned: make(map[cid.Cid]bool),
	}
}

// GetBlock retrieves a block by CID.
func (bs *BlockStore) GetBlock(ctx context.Context, c cid.Cid) ([]byte, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	key := c.String()
	_, reader, err := bs.store.Read("blocks", key)
	if err != nil {
		return nil, fmt.Errorf("block not found: %s", c)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
	}

	return buf.Bytes(), nil
}

// PutBlock stores a block with the given CID.
func (bs *BlockStore) PutBlock(ctx context.Context, c cid.Cid, data []byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := c.String()
	reader := bytes.NewReader(data)
	_, err := bs.store.Write("blocks", key, reader)
	if err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}

	return nil
}

// HasBlock checks if a block exists locally.
func (bs *BlockStore) HasBlock(c cid.Cid) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	key := c.String()
	return bs.store.Has("blocks", key)
}

// DeleteBlock removes a block from storage.
func (bs *BlockStore) DeleteBlock(c cid.Cid) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	key := c.String()
	return bs.store.Delete("blocks", key)
}

// Pin marks a block as pinned (must be retained).
func (bs *BlockStore) Pin(c cid.Cid) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.pinned[c] = true
	return nil
}

// Unpin removes the pinned status from a block.
func (bs *BlockStore) Unpin(c cid.Cid) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	delete(bs.pinned, c)
	return nil
}

// IsPinned checks if a block is pinned.
func (bs *BlockStore) IsPinned(c cid.Cid) bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return bs.pinned[c]
}

// ListPinned returns all pinned CIDs.
func (bs *BlockStore) ListPinned() []cid.Cid {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	result := make([]cid.Cid, 0, len(bs.pinned))
	for c := range bs.pinned {
		result = append(result, c)
	}
	return result
}

// AllKeysChan returns a channel that yields all stored CIDs.
func (bs *BlockStore) AllKeysChan(ctx context.Context) <-chan cid.Cid {
	ch := make(chan cid.Cid)
	go func() {
		defer close(ch)
		// Note: This is a simplified implementation
		// A full implementation would iterate over the storage directory
	}()
	return ch
}

// Adapts BlockStore to the mcp.BlockExchanger interface.
type blockExchangerAdapter struct {
	*BlockStore
}

// NewBlockExchangerAdapter wraps a BlockStore as a BlockExchanger.
func NewBlockExchangerAdapter(bs *BlockStore) mcp.BlockExchanger {
	return &blockExchangerAdapter{BlockStore: bs}
}

func (a *blockExchangerAdapter) Start() error { return nil }
func (a *blockExchangerAdapter) Stop() error  { return nil }

func (a *blockExchangerAdapter) GetBlock(ctx context.Context, c cid.Cid) ([]byte, error) {
	return a.BlockStore.GetBlock(ctx, c)
}

func (a *blockExchangerAdapter) AddBlock(ctx context.Context, c cid.Cid, block []byte) error {
	return a.BlockStore.PutBlock(ctx, c, block)
}

func (a *blockExchangerAdapter) HasBlock(c cid.Cid) bool {
	return a.BlockStore.HasBlock(c)
}

func (a *blockExchangerAdapter) SetDelegate(delegate mcp.BlockReceiver) {
	// Not used in this implementation
}
