package cid

import (
	"fmt"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// CID represents a Content Identifier interface
type CID interface {
	// String returns the string representation of the CID
	String() string

	// Bytes returns the byte representation of the CID
	Bytes() []byte

	// Hash returns the multihash of the CID
	Hash() mh.Multihash

	// Version returns the CID version
	Version() uint64

	// Codec returns the codec of the CID
	Codec() uint64

	// Equals checks if two CIDs are equal
	Equals(other CID) bool

	// Validate validates the CID
	Validate() error
}

// cidWrapper wraps an IPFS CID
type cidWrapper struct {
	cid cid.Cid
}

// CIDConfig holds configuration for CID generation
type CIDConfig struct {
	Version int
	Codec   uint64
	Hash    uint64
}

// DefaultCIDConfig returns the default CID configuration
func DefaultCIDConfig() CIDConfig {
	return CIDConfig{
		Version: 1,
		Codec:   cid.Raw,
		Hash:    mh.SHA2_256,
	}
}

// NewCID creates a new CID from raw data
func NewCID(data []byte, cfg CIDConfig) (CID, error) {
	// Generate multihash
	hash, err := mh.Sum(data, cfg.Hash, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate multihash: %w", err)
	}

	// Create CID
	var c cid.Cid
	switch cfg.Version {
	case 0:
		c = cid.NewCidV0(hash)
	case 1:
		c = cid.NewCidV1(cfg.Codec, hash)
	default:
		return nil, fmt.Errorf("unsupported CID version: %d", cfg.Version)
	}

	return &cidWrapper{cid: c}, nil
}

// CIDFromString creates a CID from a string
func CIDFromString(s string) (CID, error) {
	c, err := cid.Decode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CID: %w", err)
	}
	return &cidWrapper{cid: c}, nil
}

// CIDFromBytes creates a CID from bytes
func CIDFromBytes(b []byte) (CID, error) {
	c, err := cid.Cast(b)
	if err != nil {
		return nil, fmt.Errorf("failed to cast CID: %w", err)
	}
	return &cidWrapper{cid: c}, nil
}

func (c *cidWrapper) String() string {
	return c.cid.String()
}

func (c *cidWrapper) Bytes() []byte {
	return c.cid.Bytes()
}

func (c *cidWrapper) Hash() mh.Multihash {
	return c.cid.Hash()
}

func (c *cidWrapper) Version() uint64 {
	return c.cid.Version()
}

func (c *cidWrapper) Codec() uint64 {
	return c.cid.Prefix().Codec
}

func (c *cidWrapper) Equals(other CID) bool {
	otherWrapper, ok := other.(*cidWrapper)
	if !ok {
		return false
	}
	return c.cid.Equals(otherWrapper.cid)
}

func (c *cidWrapper) Validate() error {
	// Basic validation
	if c.cid == cid.Undef {
		return fmt.Errorf("CID is undefined")
	}
	return nil
}

// Hasher provides hashing functionality for content addressing
type Hasher interface {
	// Hash hashes data and returns the multihash
	Hash(data []byte) (mh.Multihash, error)

	// HashCode returns the hash function code
	HashCode() uint64
}

// hasherImpl implements Hasher
type hasherImpl struct {
	code uint64
}

// NewHasher creates a new hasher with the specified hash function
func NewHasher(code uint64) Hasher {
	return &hasherImpl{code: code}
}

func (h *hasherImpl) Hash(data []byte) (mh.Multihash, error) {
	return mh.Sum(data, h.code, -1)
}

func (h *hasherImpl) HashCode() uint64 {
	return h.code
}
