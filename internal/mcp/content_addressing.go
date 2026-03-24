package mcp

import (
	"fmt"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// ContentAddresser represents a content addressing interface using CID and multihash.
type ContentAddresser interface {
	// Hash data and return a CID.
	Hash(data []byte) cid.Cid

	// Convert CID to string.
	Encode(c cid.Cid) string

	// Parse string to CID.
	Decode(s string) (cid.Cid, error)

	// Get the hash function used.
	HashFunc() uint64

	// Validate if a CID is valid.
	Validate(c cid.Cid) error
}

// ipfsContentAddresser is the implementation of ContentAddresser using IPFS standards.
type ipfsContentAddresser struct {
	hashFunc uint64
	mhType   uint64
}

// NewIPFSContentAddresser creates a new IPFS content addresser with SHA2-256.
func NewIPFSContentAddresser() ContentAddresser {
	return &ipfsContentAddresser{
		hashFunc: mh.SHA2_256,
		mhType:   mh.SHA2_256,
	}
}

// Hash data and return a CID.
func (c *ipfsContentAddresser) Hash(data []byte) cid.Cid {
	// Generate multihash
	mhHash, err := mh.Sum(data, c.mhType, -1)
	if err != nil {
		// Should not happen with valid hash function
		panic(fmt.Sprintf("failed to compute multihash: %v", err))
	}

	// Create CIDv1 with Raw codec
	return cid.NewCidV1(cid.Raw, mhHash)
}

// Convert CID to string.
func (c *ipfsContentAddresser) Encode(cid cid.Cid) string {
	return cid.String()
}

// Parse string to CID.
func (c *ipfsContentAddresser) Decode(s string) (cid.Cid, error) {
	return cid.Decode(s)
}

// Get the hash function used.
func (c *ipfsContentAddresser) HashFunc() uint64 {
	return c.hashFunc
}

// Validate if a CID is valid.
func (c *ipfsContentAddresser) Validate(cid cid.Cid) error {
	if cid.Prefix().MhType != c.mhType {
		return fmt.Errorf("invalid multihash type: expected %d, got %d", c.mhType, cid.Prefix().MhType)
	}
	if cid.Version() != 1 {
		return fmt.Errorf("only CIDv1 supported, got version %d", cid.Version())
	}
	return nil
}
