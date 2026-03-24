package dag

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// Node represents a node in the Merkle DAG
type Node interface {
	// CID returns the CID of the node
	CID() cid.Cid

	// RawData returns the raw data of the node
	RawData() []byte

	// Links returns the links to other nodes
	Links() []Link

	// AddLink adds a link to another node
	AddLink(name string, target cid.Cid) error

	// RemoveLink removes a link
	RemoveLink(name string) error
}

// Link represents a link between nodes
type Link struct {
	Name string
	Cid  cid.Cid
	Size uint64
}

// Builder builds Merkle DAG nodes
type Builder interface {
	// BuildNode builds a node from data
	BuildNode(data []byte) (Node, error)

	// BuildNodeWithLinks builds a node with links
	BuildNodeWithLinks(data []byte, links []Link) (Node, error)

	// BuildTree builds a tree structure from a list of nodes
	BuildTree(nodes []Node) (Node, error)
}

// Resolver resolves Merkle DAG nodes
type Resolver interface {
	// Resolve resolves a path in the DAG
	Resolve(ctx context.Context, root cid.Cid, path string) (Node, error)

	// Get retrieves a node by its CID
	Get(ctx context.Context, c cid.Cid) (Node, error)

	// Add adds a node to the DAG
	Add(ctx context.Context, node Node) error

	// Remove removes a node from the DAG
	Remove(ctx context.Context, c cid.Cid) error
}

// dagNode implements Node
type dagNode struct {
	cid   cid.Cid
	data  []byte
	links []Link
}

// dagBuilder implements Builder
type dagBuilder struct {
	linkSystem ipld.LinkSystem
}

// dagResolver implements Resolver
type dagResolver struct {
	nodes map[cid.Cid]Node
}

// NewBuilder creates a new DAG builder
func NewBuilder(linkSystem ipld.LinkSystem) Builder {
	return &dagBuilder{
		linkSystem: linkSystem,
	}
}

// NewResolver creates a new DAG resolver
func NewResolver() Resolver {
	return &dagResolver{
		nodes: make(map[cid.Cid]Node),
	}
}

func (n *dagNode) CID() cid.Cid {
	return n.cid
}

func (n *dagNode) RawData() []byte {
	return n.data
}

func (n *dagNode) Links() []Link {
	return n.links
}

func (n *dagNode) AddLink(name string, target cid.Cid) error {
	// Check if link already exists
	for _, link := range n.links {
		if link.Name == name {
			return fmt.Errorf("link already exists: %s", name)
		}
	}

	n.links = append(n.links, Link{
		Name: name,
		Cid:  target,
	})
	return nil
}

func (n *dagNode) RemoveLink(name string) error {
	for i, link := range n.links {
		if link.Name == name {
			n.links = append(n.links[:i], n.links[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("link not found: %s", name)
}

func (b *dagBuilder) BuildNode(data []byte) (Node, error) {
	// Create a basic node
	// In practice, you would use IPLD to create a proper node
	node := &dagNode{
		data: data,
	}

	// Generate CID from data
	// This is simplified - in practice you would use proper content addressing
	node.cid = cid.Undef

	return node, nil
}

func (b *dagBuilder) BuildNodeWithLinks(data []byte, links []Link) (Node, error) {
	node, err := b.BuildNode(data)
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if err := node.AddLink(link.Name, link.Cid); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (b *dagBuilder) BuildTree(nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes provided")
	}

	// Create a root node that links to all other nodes
	rootData := []byte("root")
	root, err := b.BuildNode(rootData)
	if err != nil {
		return nil, err
	}

	for i, node := range nodes {
		linkName := fmt.Sprintf("child-%d", i)
		if err := root.AddLink(linkName, node.CID()); err != nil {
			return nil, err
		}
	}

	return root, nil
}

func (r *dagResolver) Resolve(ctx context.Context, root cid.Cid, path string) (Node, error) {
	// Get the root node
	node, err := r.Get(ctx, root)
	if err != nil {
		return nil, err
	}

	// Simplified path resolution
	// In practice, you would parse the path and traverse the DAG
	return node, nil
}

func (r *dagResolver) Get(ctx context.Context, c cid.Cid) (Node, error) {
	node, exists := r.nodes[c]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", c)
	}
	return node, nil
}

func (r *dagResolver) Add(ctx context.Context, node Node) error {
	r.nodes[node.CID()] = node
	return nil
}

func (r *dagResolver) Remove(ctx context.Context, c cid.Cid) error {
	delete(r.nodes, c)
	return nil
}

// LinkSystem provides a link system for IPLD operations
type LinkSystem struct {
	linkSystem ipld.LinkSystem
}

// NewLinkSystem creates a new link system
func NewLinkSystem() *LinkSystem {
	// Create a basic link system
	// In practice, you would configure it with proper storage
	ls := cidlink.DefaultLinkSystem()
	return &LinkSystem{
		linkSystem: ls,
	}
}
