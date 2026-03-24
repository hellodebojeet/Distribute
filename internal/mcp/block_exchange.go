package mcp

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
)

// BlockExchanger represents a block exchange interface for distributing data across peers.
type BlockExchanger interface {
	// Start the block exchanger.
	Start() error

	// Stop the block exchanger.
	Stop() error

	// GetBlock requests a block from the network.
	GetBlock(ctx context.Context, k cid.Cid) ([]byte, error)

	// AddBlock provides a block to the network.
	AddBlock(ctx context.Context, k cid.Cid, block []byte) error

	// HasBlock checks if we have a block locally.
	HasBlock(k cid.Cid) bool

	// SetDelegate sets the block receiver delegate.
	SetDelegate(delegate BlockReceiver)
}

// BlockReceiver represents a delegate for receiving blocks.
type BlockReceiver interface {
	// BlockReceived is called when a block is received.
	BlockReceived(cid.Cid, []byte) error
}

// simpleBlockExchanger is a basic implementation that can be replaced with a full bitswap implementation.
type simpleBlockExchanger struct {
	blocks   map[cid.Cid][]byte
	delegate BlockReceiver
}

// NewSimpleBlockExchanger creates a new in-memory block exchanger.
func NewSimpleBlockExchanger() BlockExchanger {
	return &simpleBlockExchanger{
		blocks: make(map[cid.Cid][]byte),
	}
}

// Start the block exchanger.
func (b *simpleBlockExchanger) Start() error {
	return nil
}

// Stop the block exchanger.
func (b *simpleBlockExchanger) Stop() error {
	return nil
}

// GetBlock requests a block from the network.
func (b *simpleBlockExchanger) GetBlock(ctx context.Context, k cid.Cid) ([]byte, error) {
	if block, exists := b.blocks[k]; exists {
		return block, nil
	}
	return nil, fmt.Errorf("block not found: %s", k)
}

// AddBlock provides a block to the network.
func (b *simpleBlockExchanger) AddBlock(ctx context.Context, k cid.Cid, block []byte) error {
	b.blocks[k] = block
	if b.delegate != nil {
		return b.delegate.BlockReceived(k, block)
	}
	return nil
}

// HasBlock checks if we have a block locally.
func (b *simpleBlockExchanger) HasBlock(k cid.Cid) bool {
	_, exists := b.blocks[k]
	return exists
}

// SetDelegate sets the block receiver delegate.
func (b *simpleBlockExchanger) SetDelegate(delegate BlockReceiver) {
	b.delegate = delegate
}
