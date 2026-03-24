package replication

import (
	"fmt"
	"sync"
	"time"

	"github.com/hellodebojeet/Distribute/metadata"
)

// ReplicationManager handles chunk replication across nodes.
type ReplicationManager interface {
	ReplicateChunk(fileID string, chunkID string, data []byte, locations []string) error
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
func (r *ReplicationManagerImpl) ReplicateChunk(fileID string, chunkID string, data []byte, locations []string) error {
	if len(locations) == 0 {
		return fmt.Errorf("no locations specified for chunk %s", chunkID)
	}

	var failedLocations []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Retry configuration
	maxRetries := r.maxRetries
	retryDelay := time.Second

	for _, nodeID := range locations {
		// Attempt replication with retries
		success := false
		for attempt := 0; attempt <= maxRetries && !success; attempt++ {
			if attempt > 0 {
				// Wait before retry
				time.Sleep(retryDelay)
				// Exponential backoff
				retryDelay *= 2
			}

			// Get node info from metadata service
			node, err := r.metadataClient.GetNode(nodeID)
			if err != nil {
				if attempt == maxRetries {
					mu.Lock()
					failedLocations = append(failedLocations, nodeID)
					mu.Unlock()
				}
				continue
			}

			// Check if node is alive
			if !node.IsAlive {
				if attempt == maxRetries {
					mu.Lock()
					failedLocations = append(failedLocations, nodeID)
					mu.Unlock()
				}
				continue
			}

			// TODO: Implement actual RPC call to replicate chunk to this node
			// This is a placeholder - in real implementation, you would send the chunk data to the peer
			// using the existing p2p transport layer.
			//
			// Placeholder: In a full implementation, we would:
			// 1. Establish connection to the node using its address
			// 2. Send a ReplicateChunk message with the chunkID and data
			// 3. Wait for acknowledgment
			//
			// For this implementation, we'll mark as successful if node is alive
			// The actual implementation would need to be integrated with the FileServer's transport

			// Simulate successful replication for now
			success = true
		}
	}

	wg.Wait()

	if len(failedLocations) > 0 {
		// Update metadata to mark chunk as under-replicated
		// This will trigger background repair process
		if err := r.metadataClient.MarkUnderReplicated(chunkID, fileID); err != nil {
			return fmt.Errorf("failed to replicate chunk %s to %d nodes and failed to mark as under-replicated: %v", chunkID, len(failedLocations), err)
		}
		return fmt.Errorf("failed to replicate chunk %s to %d nodes: %v", chunkID, len(failedLocations), failedLocations)
	}

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
