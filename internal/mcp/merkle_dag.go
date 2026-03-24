package mcp

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
)

// MerkleDAG represents a Merkle DAG interface for IPLD.
type MerkleDAG interface {
	// AddNode adds a node to the DAG and returns its CID.
	AddNode(ctx context.Context, nodeData []byte) (cid.Cid, error)

	// GetNode retrieves a node by its CID.
	GetNode(ctx context.Context, cid cid.Cid) ([]byte, error)

	// AddLink adds a link from one node to another.
	AddLink(ctx context.Context, from cid.Cid, name string, to cid.Cid) error

	// GetLinks retrieves all links from a node.
	GetLinks(ctx context.Context, from cid.Cid) ([]Link, error)

	// RemoveNode removes a node from the DAG.
	RemoveNode(ctx context.Context, cid cid.Cid) error
}

// Link represents a link in the Merkle DAG.
type Link struct {
	Name string
	CID  cid.Cid
}

// SimpleMerkleDAG is a basic implementation that can be replaced with a full IPLD implementation.
type SimpleMerkleDAG struct {
	nodes map[cid.Cid][]byte
	links map[cid.Cid]map[string]cid.Cid
	ca    ContentAddresser
}

// NewSimpleMerkleDAG creates a new in-memory Merkle DAG.
func NewSimpleMerkleDAG(ca ContentAddresser) MerkleDAG {
	return &SimpleMerkleDAG{
		nodes: make(map[cid.Cid][]byte),
		links: make(map[cid.Cid]map[string]cid.Cid),
		ca:    ca,
	}
}

// AddNode adds a node to the DAG and returns its CID.
func (d *SimpleMerkleDAG) AddNode(ctx context.Context, nodeData []byte) (cid.Cid, error) {
	// Use content addresser to generate CID from node data
	c := d.ca.Hash(nodeData)

	d.nodes[c] = nodeData
	return c, nil
}

// GetNode retrieves a node by its CID.
func (d *SimpleMerkleDAG) GetNode(ctx context.Context, c cid.Cid) ([]byte, error) {
	if data, exists := d.nodes[c]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("node not found: %s", c)
}

// AddLink adds a link from one node to another.
func (d *SimpleMerkleDAG) AddLink(ctx context.Context, from cid.Cid, name string, to cid.Cid) error {
	if _, exists := d.nodes[from]; !exists {
		return fmt.Errorf("source node not found: %s", from)
	}
	if _, exists := d.nodes[to]; !exists {
		return fmt.Errorf("target node not found: %s", to)
	}

	if d.links[from] == nil {
		d.links[from] = make(map[string]cid.Cid)
	}
	d.links[from][name] = to
	return nil
}

// GetLinks retrieves all links from a node.
func (d *SimpleMerkleDAG) GetLinks(ctx context.Context, from cid.Cid) ([]Link, error) {
	if _, exists := d.nodes[from]; !exists {
		return nil, fmt.Errorf("node not found: %s", from)
	}

	links := []Link{}
	for name, to := range d.links[from] {
		links = append(links, Link{Name: name, CID: to})
	}
	return links, nil
}

// RemoveNode removes a node from the DAG.
func (d *SimpleMerkleDAG) RemoveNode(ctx context.Context, c cid.Cid) error {
	if _, exists := d.nodes[c]; !exists {
		return fmt.Errorf("node not found: %s", c)
	}

	delete(d.nodes, c)
	delete(d.links, c)

	// Remove links to this node from other nodes
	for _, linkMap := range d.links {
		for name, target := range linkMap {
			if target == c {
				delete(linkMap, name)
			}
		}
	}
	return nil
}
