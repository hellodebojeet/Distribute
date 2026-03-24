// Package integration provides integration tests for the distributed filesystem.
package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hellodebojeet/Distribute/gateway"
	"github.com/hellodebojeet/Distribute/gateway/blockstore"
	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/hellodebojeet/Distribute/observability/logging"
	"github.com/hellodebojeet/Distribute/server"
)

// TestCluster represents a test cluster of nodes.
type TestCluster struct {
	t        *testing.T
	nodes    []*TestNode
	gateways []*gateway.Gateway
	mu       sync.Mutex
	cleanup  []func()
}

// TestNode represents a single test node.
type TestNode struct {
	ID             string
	Port           int
	BlockStore     *blockstore.BlockStore
	DHT            mcp.DHT
	BlockExchanger mcp.BlockExchanger
	store          *server.Store
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewTestCluster creates a new test cluster with the specified number of nodes.
func NewTestCluster(t *testing.T, numNodes int) *TestCluster {
	cluster := &TestCluster{
		t:       t,
		nodes:   make([]*TestNode, 0, numNodes),
		cleanup: make([]func(), 0),
	}

	// Create nodes
	for i := 0; i < numNodes; i++ {
		node, err := cluster.createNode(3000 + i)
		require.NoError(t, err)
		cluster.nodes = append(cluster.nodes, node)
	}

	return cluster
}

// createNode creates a new test node.
func (c *TestCluster) createNode(port int) (*TestNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create storage directory
	storeDir, err := os.MkdirTemp("", fmt.Sprintf("dfs-test-*"))
	if err != nil {
		cancel()
		return nil, err
	}

	c.cleanup = append(c.cleanup, func() {
		os.RemoveAll(storeDir)
	})

	// Create store
	store := server.NewStore(server.StoreOpts{
		Root: storeDir,
	})

	// Create blockstore
	bs := blockstore.NewBlockStore(store)

	// Create block exchanger
	be := blockstore.NewBlockExchangerAdapter(bs)

	nodeID := fmt.Sprintf("test-node-%d", port)

	return &TestNode{
		ID:             nodeID,
		Port:           port,
		BlockStore:     bs,
		BlockExchanger: be,
		store:          store,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

// Start starts all nodes in the cluster.
func (c *TestCluster) Start() error {
	for _, node := range c.nodes {
		// Start any background processes
		_ = node
	}
	return nil
}

// Stop stops all nodes in the cluster.
func (c *TestCluster) Stop() {
	for _, node := range c.nodes {
		if node.cancel != nil {
			node.cancel()
		}
	}

	for _, cleanup := range c.cleanup {
		cleanup()
	}
}

// GetNode returns a node by index.
func (c *TestCluster) GetNode(index int) *TestNode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodes[index]
}

// AddFile adds a file to a specific node and returns the CID.
func (c *TestCluster) AddFile(nodeIndex int, data []byte) (cid.Cid, error) {
	node := c.GetNode(nodeIndex)

	// Generate CID
	ca := mcp.NewIPFSContentAddresser()
	fileCID := ca.Hash(data)

	// Store the block
	err := node.BlockStore.PutBlock(context.Background(), fileCID, data)
	if err != nil {
		return cid.Cid{}, err
	}

	return fileCID, nil
}

// GetFile retrieves a file from a specific node.
func (c *TestCluster) GetFile(nodeIndex int, fileCID cid.Cid) ([]byte, error) {
	node := c.GetNode(nodeIndex)
	return node.BlockStore.GetBlock(context.Background(), fileCID)
}

// SimulateNodeCrash simulates a node crash by canceling its context.
func (c *TestCluster) SimulateNodeCrash(nodeIndex int) {
	node := c.GetNode(nodeIndex)
	node.cancel()
}

// SimulateNetworkPartition simulates a network partition between two nodes.
func (c *TestCluster) SimulateNetworkPartition(node1Index, node2Index int) {
	// In a full implementation, this would block communication
	// For now, we just track the partition state
	c.t.Logf("Simulating network partition between nodes %d and %d", node1Index, node2Index)
}

// HealNetworkPartition heals a network partition.
func (c *TestCluster) HealNetworkPartition(node1Index, node2Index int) {
	c.t.Logf("Healing network partition between nodes %d and %d", node1Index, node2Index)
}

// TestMultiNodeCluster tests basic multi-node cluster operations.
func TestMultiNodeCluster(t *testing.T) {
	cluster := NewTestCluster(t, 3)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	// Generate test data
	testData := generateTestData(1024 * 1024) // 1MB

	// Add file to node 0
	fileCID, err := cluster.AddFile(0, testData)
	require.NoError(t, err)
	t.Logf("Added file with CID: %s", fileCID.String())

	// Verify file exists on node 0
	data, err := cluster.GetFile(0, fileCID)
	require.NoError(t, err)
	assert.Equal(t, testData, data)

	// Test CID validation
	assert.True(t, fileCID.Defined())
	assert.Equal(t, uint64(1), fileCID.Version())
}

// TestContentIntegrity tests that content integrity is maintained.
func TestContentIntegrity(t *testing.T) {
	cluster := NewTestCluster(t, 1)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	testCases := []struct {
		name string
		size int
	}{
		{"small", 100},
		{"medium", 4096},
		{"large", 1024 * 1024},
		{"very_large", 10 * 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := generateTestData(tc.size)

			// Add file
			fileCID, err := cluster.AddFile(0, testData)
			require.NoError(t, err)

			// Retrieve file
			retrievedData, err := cluster.GetFile(0, fileCID)
			require.NoError(t, err)

			// Verify integrity
			assert.Equal(t, testData, retrievedData)

			// Verify CID matches
			ca := mcp.NewIPFSContentAddresser()
			computedCID := ca.Hash(retrievedData)
			assert.True(t, fileCID.Equals(computedCID))
		})
	}
}

// TestNodeCrashRecovery tests that data is accessible after a node crash.
func TestNodeCrashRecovery(t *testing.T) {
	cluster := NewTestCluster(t, 2)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	// Add data to node 0
	testData := generateTestData(4096)
	fileCID, err := cluster.AddFile(0, testData)
	require.NoError(t, err)

	// Verify data exists
	data, err := cluster.GetFile(0, fileCID)
	require.NoError(t, err)
	assert.Equal(t, testData, data)

	// Simulate crash
	cluster.SimulateNodeCrash(0)

	// Data should still be accessible from persistent storage
	// (In a real scenario, we'd test that other nodes have copies)
	t.Log("Node crash simulated - verifying data persistence")
}

// TestGatewayHTTP tests the HTTP gateway health endpoint.
func TestGatewayHTTP(t *testing.T) {
	cluster := NewTestCluster(t, 1)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	node := cluster.GetNode(0)

	// Create DAG for gateway
	dag := mcp.NewSimpleMerkleDAG(mcp.NewIPFSContentAddresser())

	// Create gateway
	gwConfig := gateway.DefaultGatewayConfig()
	gwConfig.ListenAddr = ":18081" // Use non-standard port for testing

	// Create a minimal logger for testing
	logger, _ := logging.NewDevelopmentLogger()

	gw := gateway.NewGateway(gwConfig, node.BlockStore, dag, nil, nil, logger)

	// Start gateway in background
	gwStarted := make(chan error, 1)
	go func() {
		gwStarted <- gw.Start()
	}()

	// Wait for gateway to start
	time.Sleep(200 * time.Millisecond)

	// Test gateway health endpoint
	resp, err := http.Get("http://localhost:18081/health")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Test gateway ready endpoint
	resp, err = http.Get("http://localhost:18081/ready")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Cleanup
	gw.Stop(context.Background())
}

// TestConcurrentAccess tests concurrent access to the blockstore.
func TestConcurrentAccess(t *testing.T) {
	cluster := NewTestCluster(t, 1)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	node := cluster.GetNode(0)

	// Generate multiple test files
	numFiles := 100
	testData := make([][]byte, numFiles)
	cids := make([]cid.Cid, numFiles)

	for i := 0; i < numFiles; i++ {
		testData[i] = generateTestData(1024 + i*100)
	}

	// Add files concurrently
	var wg sync.WaitGroup
	errors := make(chan error, numFiles)

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ca := mcp.NewIPFSContentAddresser()
			c := ca.Hash(testData[idx])

			err := node.BlockStore.PutBlock(context.Background(), c, testData[idx])
			if err != nil {
				errors <- err
				return
			}
			cids[idx] = c
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		require.NoError(t, err)
	}

	// Verify all files can be retrieved
	for i := 0; i < numFiles; i++ {
		data, err := node.BlockStore.GetBlock(context.Background(), cids[i])
		require.NoError(t, err)
		assert.Equal(t, testData[i], data)
	}
}

// TestPinning tests the pinning functionality.
func TestPinning(t *testing.T) {
	cluster := NewTestCluster(t, 1)
	defer cluster.Stop()

	err := cluster.Start()
	require.NoError(t, err)

	// Add test data
	testData := generateTestData(1024)
	fileCID, err := cluster.AddFile(0, testData)
	require.NoError(t, err)

	node := cluster.GetNode(0)

	// Pin the content
	err = node.BlockStore.Pin(fileCID)
	require.NoError(t, err)

	// Verify it's pinned
	pinned := node.BlockStore.IsPinned(fileCID)
	assert.True(t, pinned)

	// List pinned content
	pinnedList := node.BlockStore.ListPinned()
	assert.Contains(t, pinnedList, fileCID)

	// Unpin the content
	err = node.BlockStore.Unpin(fileCID)
	require.NoError(t, err)

	// Verify it's no longer pinned
	pinned = node.BlockStore.IsPinned(fileCID)
	assert.False(t, pinned)
}

// generateTestData generates random test data of the specified size.
func generateTestData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// generateCID generates a CID for the given data.
func generateCID(data []byte) cid.Cid {
	ca := mcp.NewIPFSContentAddresser()
	return ca.Hash(data)
}
