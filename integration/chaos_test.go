// Package integration provides chaos engineering tests for failure injection.
package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hellodebojeet/Distribute/gateway/blockstore"
	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/hellodebojeet/Distribute/server"
)

// ChaosTestCluster extends TestCluster with chaos capabilities.
type ChaosTestCluster struct {
	t       *testing.T
	nodes   []*ChaosNode
	mu      sync.Mutex
	chaosCh chan ChaosEvent
	stopped bool
}

// ChaosNode represents a node that can be subjected to chaos.
type ChaosNode struct {
	ID         string
	BlockStore *blockstore.BlockStore
	store      *server.Store
	ctx        context.Context
	cancel     context.CancelFunc
	isAlive    bool
	mu         sync.RWMutex
}

// ChaosEvent represents a chaos event.
type ChaosEvent struct {
	Type      ChaosEventType
	NodeID    string
	Timestamp time.Time
}

// ChaosEventType represents the type of chaos event.
type ChaosEventType int

const (
	EventTypeCrash ChaosEventType = iota
	EventTypeRecover
	EventTypeNetworkPartition
	EventTypeNetworkHeal
	EventTypeDiskFull
	EventTypeSlowDisk
)

// NewChaosCluster creates a new chaos test cluster.
func NewChaosCluster(t *testing.T, numNodes int) *ChaosTestCluster {
	cluster := &ChaosTestCluster{
		t:       t,
		nodes:   make([]*ChaosNode, 0, numNodes),
		chaosCh: make(chan ChaosEvent, 100),
	}

	for i := 0; i < numNodes; i++ {
		node, err := cluster.createNode(i)
		require.NoError(t, err)
		cluster.nodes = append(cluster.nodes, node)
	}

	return cluster
}

func (c *ChaosTestCluster) createNode(index int) (*ChaosNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	storeDir, err := os.MkdirTemp("", fmt.Sprintf("chaos-node-%d-*", index))
	if err != nil {
		cancel()
		return nil, err
	}

	store := server.NewStore(server.StoreOpts{
		Root: storeDir,
	})

	bs := blockstore.NewBlockStore(store)

	nodeID := fmt.Sprintf("chaos-node-%d", index)

	return &ChaosNode{
		ID:         nodeID,
		BlockStore: bs,
		store:      store,
		ctx:        ctx,
		cancel:     cancel,
		isAlive:    true,
	}, nil
}

// GetNode returns a node by index.
func (c *ChaosTestCluster) GetNode(index int) *ChaosNode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodes[index]
}

// CrashNode simulates a node crash.
func (c *ChaosTestCluster) CrashNode(nodeIndex int) {
	node := c.GetNode(nodeIndex)
	node.mu.Lock()
	defer node.mu.Unlock()

	node.isAlive = false
	node.cancel()

	c.chaosCh <- ChaosEvent{
		Type:      EventTypeCrash,
		NodeID:    node.ID,
		Timestamp: time.Now(),
	}

	c.t.Logf("Node %s crashed", node.ID)
}

// RecoverNode simulates node recovery.
func (c *ChaosTestCluster) RecoverNode(nodeIndex int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	node := c.nodes[nodeIndex]
	node.mu.Lock()
	defer node.mu.Unlock()

	// Create new context
	ctx, cancel := context.WithCancel(context.Background())
	node.ctx = ctx
	node.cancel = cancel
	node.isAlive = true

	c.chaosCh <- ChaosEvent{
		Type:      EventTypeRecover,
		NodeID:    node.ID,
		Timestamp: time.Now(),
	}

	c.t.Logf("Node %s recovered", node.ID)
	return nil
}

// AddData adds data to a node.
func (c *ChaosTestCluster) AddData(nodeIndex int, data []byte) (cid.Cid, error) {
	node := c.GetNode(nodeIndex)

	node.mu.RLock()
	if !node.isAlive {
		node.mu.RUnlock()
		return cid.Cid{}, fmt.Errorf("node %s is not alive", node.ID)
	}
	node.mu.RUnlock()

	ca := mcp.NewIPFSContentAddresser()
	fileCID := ca.Hash(data)

	err := node.BlockStore.PutBlock(context.Background(), fileCID, data)
	if err != nil {
		return cid.Cid{}, err
	}

	return fileCID, nil
}

// GetData retrieves data from a node.
func (c *ChaosTestCluster) GetData(nodeIndex int, fileCID cid.Cid) ([]byte, error) {
	node := c.GetNode(nodeIndex)

	node.mu.RLock()
	if !node.isAlive {
		node.mu.RUnlock()
		return nil, fmt.Errorf("node %s is not alive", node.ID)
	}
	node.mu.RUnlock()

	return node.BlockStore.GetBlock(context.Background(), fileCID)
}

// Stop stops all nodes.
func (c *ChaosTestCluster) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}
	c.stopped = true

	for _, node := range c.nodes {
		node.cancel()
	}
	close(c.chaosCh)
}

// GetChaosEvents returns the chaos event channel.
func (c *ChaosTestCluster) GetChaosEvents() <-chan ChaosEvent {
	return c.chaosCh
}

// TestCrashRecovery tests node crash and recovery.
func TestCrashRecovery(t *testing.T) {
	cluster := NewChaosCluster(t, 3)
	defer cluster.Stop()

	// Add data to node 0
	testData := generateChaosTestData(4096)
	fileCID, err := cluster.AddData(0, testData)
	require.NoError(t, err)

	// Verify data exists
	data, err := cluster.GetData(0, fileCID)
	require.NoError(t, err)
	assert.Equal(t, testData, data)

	// Crash node 0
	cluster.CrashNode(0)

	// Try to access data - should fail
	_, err = cluster.GetData(0, fileCID)
	assert.Error(t, err)

	// Recover node 0
	err = cluster.RecoverNode(0)
	require.NoError(t, err)

	// Data should still be accessible from storage
	data, err = cluster.GetData(0, fileCID)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

// TestConcurrentCrash tests multiple concurrent crashes.
func TestConcurrentCrash(t *testing.T) {
	cluster := NewChaosCluster(t, 5)
	defer cluster.Stop()

	// Add data to all nodes
	cids := make([]cid.Cid, 5)
	testData := generateChaosTestData(1024)

	for i := 0; i < 5; i++ {
		cid, err := cluster.AddData(i, testData)
		require.NoError(t, err)
		cids[i] = cid
	}

	// Crash nodes concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			time.Sleep(time.Duration(idx*100) * time.Millisecond)
			cluster.CrashNode(idx)
		}(i)
	}
	wg.Wait()

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Recover all nodes
	for i := 0; i < 5; i++ {
		err := cluster.RecoverNode(i)
		require.NoError(t, err)
	}

	// Verify data is still accessible
	for i := 0; i < 5; i++ {
		data, err := cluster.GetData(i, cids[i])
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	}
}

// TestChaosEvents tests that chaos events are properly recorded.
func TestChaosEvents(t *testing.T) {
	cluster := NewChaosCluster(t, 2)
	defer cluster.Stop()

	// Start event collector
	events := make([]ChaosEvent, 0)
	var eventsMu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for event := range cluster.GetChaosEvents() {
			eventsMu.Lock()
			events = append(events, event)
			eventsMu.Unlock()
		}
	}()

	// Perform some chaos
	cluster.CrashNode(0)
	time.Sleep(100 * time.Millisecond)
	cluster.RecoverNode(0)
	time.Sleep(100 * time.Millisecond)
	cluster.CrashNode(1)

	// Wait for events
	time.Sleep(500 * time.Millisecond)
	cluster.Stop()
	wg.Wait()

	// Verify events
	eventsMu.Lock()
	assert.GreaterOrEqual(t, len(events), 3, "Expected at least 3 chaos events")
	eventsMu.Unlock()
}

// TestDataPersistenceAfterCrash tests that data persists after crashes.
func TestDataPersistenceAfterCrash(t *testing.T) {
	cluster := NewChaosCluster(t, 1)
	defer cluster.Stop()

	node := cluster.GetNode(0)

	// Create multiple files
	files := make(map[cid.Cid][]byte)
	for i := 0; i < 10; i++ {
		testData := generateChaosTestData(1024 * (i + 1))
		fileCID, err := cluster.AddData(0, testData)
		require.NoError(t, err)
		files[fileCID] = testData
	}

	// Crash and recover multiple times
	for cycle := 0; cycle < 3; cycle++ {
		cluster.CrashNode(0)
		time.Sleep(100 * time.Millisecond)
		err := cluster.RecoverNode(0)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Verify all files are still accessible
	for fileCID, expectedData := range files {
		data, err := node.BlockStore.GetBlock(context.Background(), fileCID)
		require.NoError(t, err)
		assert.Equal(t, expectedData, data, "Data mismatch for CID %s", fileCID)
	}
}

// generateChaosTestData generates random test data for chaos tests.
func generateChaosTestData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}
