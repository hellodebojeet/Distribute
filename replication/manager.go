package replication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anthdm/foreverstore/metadata"
)

// ReplicationManager handles chunk replication across nodes.
type ReplicationManager interface {
	ReplicateChunk(chunkID string, data []byte, locations []string) error
	GetReplicationStatus(chunkID string) (int, int, error) // actual, expected
}

// NodeSelector defines strategy for selecting nodes for replication.
type NodeSelector interface {
	SelectNodes(fileID string, numReplicas int) ([]string, error)
}

// SimpleNodeSelector implements basic node selection using hash + successors.
type SimpleNodeSelector struct {
	metadataClient metadata.MetadataClient
}

func NewSimpleNodeSelector(client metadata.MetadataClient) *SimpleNodeSelector {
	return &SimpleNodeSelector{metadataClient: client}
}

func (s *SimpleNodeSelector) SelectNodes(fileID string, numReplicas int) ([]string, error) {
	// Get all available nodes
	nodes, err := s.metadataClient.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes for replication")
	}

	// Simple selection: use hash of fileID to pick starting node, then select N successors
	hash := hashString(fileID)
	startIndex := int(hash % uint64(len(nodes)))

	selectedNodes := make([]string, 0, numReplicas)
	for i := 0; i < len(nodes) && len(selectedNodes) < numReplicas; i++ {
		index := (startIndex + i) % len(nodes)
		selectedNodes = append(selectedNodes, nodes[index].NodeID)
	}

	return selectedNodes, nil
}

// ReplicationManagerImpl implements ReplicationManager interface.
type ReplicationManagerImpl struct {
	metadataClient metadata.MetadataClient
	nodeSelector   NodeSelector
	maxRetries     int
	timeout        time.Duration
	mu             sync.RWMutex
}

// NewReplicationManager creates a new replication manager.
func NewReplicationManager(client metadata.MetadataClient, selector NodeSelector) *ReplicationManagerImpl {
	return &ReplicationManagerImpl{
		metadataClient: client,
		nodeSelector:   selector,
		maxRetries:     3,
		timeout:        10 * time.Second,
	}
}

// ReplicateChunk replicates a chunk to specified nodes.
func (r *ReplicationManagerImpl) ReplicateChunk(chunkID string, data []byte, locations []string) error {
	if len(locations) == 0 {
		return fmt.Errorf("no locations specified for chunk %s", chunkID)
	}

	var failedLocations []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, nodeID := range locations {
		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()

			// Get node info from metadata service
			node, err := r.metadataClient.GetNode(nodeID)
			if err != nil {
				mu.Lock()
				failedLocations = append(failedLocations, nodeID)
				mu.Unlock()
				return
			}

			// TODO: Implement actual RPC call to replicate chunk to this node
			// This is a placeholder - in real implementation, you would send the chunk data to the peer
			// using the existing p2p transport layer.
			// For now, we'll just simulate success/failure based on node health.

			// Simulate replication with random success/failure for demo purposes
			if node.IsAlive {
				// In real implementation, send the chunk data to the node via RPC
				// Example: nodeTransport.SendChunk(chunkID, data)
			} else {
				// Node is not alive - mark as failed
				mu.Lock()
				failedLocations = append(failedLocations, nodeID)
				mu.Unlock()
			}
		}(nodeID)
	}

	wg.Wait()

	if len(failedLocations) > 0 {
		// Update metadata to mark chunk as under-replicated
		// This will trigger background repair process
		// In production, you might want to implement retry logic here
		return fmt.Errorf("failed to replicate chunk %s to %d nodes: %v", chunkID, len(failedLocations), failedLocations)
	}

	// Update metadata to mark chunk as successfully replicated
	// In real implementation, you would call a method like:
	// r.metadataClient.CommitChunkReplication(chunkID, locations)

	return nil
}

// GetReplicationStatus returns the current replication status for a chunk.
func (r *ReplicationManagerImpl) GetReplicationStatus(chunkID string) (int, int, error) {
	// Get current locations from metadata
	locations, err := r.metadataClient.GetChunkLocations(chunkID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get chunk locations: %w", err)
	}

	// Count live nodes that have the chunk
	var liveCount int
	for _, nodeID := range locations {
		// Get node info from metadata service
		node, err := r.metadataClient.GetNode(nodeID)
		if err != nil {
			continue
		}
		if node.IsAlive {
			liveCount++
		}
	}

	// Expected replicas should be configurable (RF)
	actual := liveCount
	expected := len(locations) // Simplified - in production you'd track desired RF

	return actual, expected, nil
}

// hashString generates a simple hash for string.
func hashString(s string) uint64 {
	var h uint64
	for i := 0; i < len(s); i++ {
		h = (h << 5) - h + uint64(s[i])
	}
	return h
}
