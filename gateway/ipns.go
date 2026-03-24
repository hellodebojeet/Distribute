// Package ipns provides IPNS (InterPlanetary Name System) support for mutable pointers.
package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
)

// IPNSRecord represents an IPNS record for mutable name resolution.
type IPNSRecord struct {
	// Value is the CID this name points to
	Value cid.Cid
	// Sequence number for record ordering
	Sequence uint64
	// Validity is when this record expires
	Validity time.Time
	// TTL is the recommended cache duration
	TTL time.Duration
	// Signature is the cryptographic signature
	Signature []byte
	// PeerID is the peer that published this record
	PeerID string
}

// IPNSResolver resolves IPNS names to CIDs.
type IPNSResolver interface {
	// Resolve resolves an IPNS name to a CID.
	Resolve(ctx context.Context, name string) (cid.Cid, error)

	// ResolveRecord returns the full IPNS record.
	ResolveRecord(ctx context.Context, name string) (*IPNSRecord, error)

	// Publish creates or updates an IPNS record.
	Publish(ctx context.Context, name string, value cid.Cid, ttl time.Duration) error

	// ListNames returns all known IPNS names.
	ListNames() []string
}

// MemoryIPNSResolver implements IPNSResolver using in-memory storage.
// In production, this would be backed by the DHT or a persistent datastore.
type MemoryIPNSResolver struct {
	mu      sync.RWMutex
	records map[string]*IPNSRecord
}

// NewMemoryIPNSResolver creates a new in-memory IPNS resolver.
func NewMemoryIPNSResolver() *MemoryIPNSResolver {
	return &MemoryIPNSResolver{
		records: make(map[string]*IPNSRecord),
	}
}

// Resolve resolves an IPNS name to a CID.
func (r *MemoryIPNSResolver) Resolve(ctx context.Context, name string) (cid.Cid, error) {
	record, err := r.ResolveRecord(ctx, name)
	if err != nil {
		return cid.Undef, err
	}
	return record.Value, nil
}

// ResolveRecord returns the full IPNS record.
func (r *MemoryIPNSResolver) ResolveRecord(ctx context.Context, name string) (*IPNSRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, exists := r.records[name]
	if !exists {
		return nil, fmt.Errorf("IPNS name not found: %s", name)
	}

	// Check if record has expired
	if !record.Validity.IsZero() && time.Now().After(record.Validity) {
		return nil, fmt.Errorf("IPNS record expired: %s", name)
	}

	return record, nil
}

// Publish creates or updates an IPNS record.
func (r *MemoryIPNSResolver) Publish(ctx context.Context, name string, value cid.Cid, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get existing record to increment sequence
	var sequence uint64
	if existing, exists := r.records[name]; exists {
		sequence = existing.Sequence + 1
	}

	// Create new record
	record := &IPNSRecord{
		Value:    value,
		Sequence: sequence,
		Validity: time.Now().Add(24 * time.Hour), // Default 24h validity
		TTL:      ttl,
		PeerID:   "", // Would be set from the publishing peer
	}

	r.records[name] = record
	return nil
}

// ListNames returns all known IPNS names.
func (r *MemoryIPNSResolver) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.records))
	for name := range r.records {
		names = append(names, name)
	}
	return names
}

// DHTIPNSResolver resolves IPNS records via the DHT.
// This is a placeholder for full DHT-backed IPNS implementation.
type DHTIPNSResolver struct {
	dht      DHTInterface
	cache    *MemoryIPNSResolver
	cacheTTL time.Duration
}

// DHTInterface is the interface for DHT operations needed by IPNS.
type DHTInterface interface {
	Provide(ctx context.Context, key cid.Cid, announce bool) error
	FindProvidersAsync(ctx context.Context, key cid.Cid, maxProviders int) <-chan PeerInfo
}

// PeerInfo represents peer information from DHT.
type PeerInfo struct {
	ID    string
	Addrs []string
}

// NewDHTIPNSResolver creates a new DHT-backed IPNS resolver.
func NewDHTIPNSResolver(dht DHTInterface) *DHTIPNSResolver {
	return &DHTIPNSResolver{
		dht:      dht,
		cache:    NewMemoryIPNSResolver(),
		cacheTTL: 5 * time.Minute,
	}
}

// Resolve resolves an IPNS name via DHT with cache fallback.
func (r *DHTIPNSResolver) Resolve(ctx context.Context, name string) (cid.Cid, error) {
	record, err := r.ResolveRecord(ctx, name)
	if err != nil {
		return cid.Undef, err
	}
	return record.Value, nil
}

// ResolveRecord returns the full IPNS record from DHT or cache.
func (r *DHTIPNSResolver) ResolveRecord(ctx context.Context, name string) (*IPNSRecord, error) {
	// First try cache
	record, err := r.cache.ResolveRecord(ctx, name)
	if err == nil {
		return record, nil
	}

	// TODO: Fetch from DHT
	// In a full implementation:
	// 1. Hash the IPNS name to get a content key
	// 2. Query DHT for providers of that key
	// 3. Fetch the IPNS record from providers
	// 4. Validate the record signature
	// 5. Cache the record

	return nil, fmt.Errorf("IPNS record not found in DHT: %s", name)
}

// Publish publishes an IPNS record to the DHT.
func (r *DHTIPNSResolver) Publish(ctx context.Context, name string, value cid.Cid, ttl time.Duration) error {
	// Update cache first
	if err := r.cache.Publish(ctx, name, value, ttl); err != nil {
		return err
	}

	// TODO: Publish to DHT
	// In a full implementation:
	// 1. Create the IPNS record with signature
	// 2. Store in DHT with appropriate key
	// 3. Announce to the network

	return nil
}

// ListNames returns all cached IPNS names.
func (r *DHTIPNSResolver) ListNames() []string {
	return r.cache.ListNames()
}
