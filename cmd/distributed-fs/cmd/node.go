package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/spf13/cobra"
)

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage storage nodes",
	Long: `Start and manage storage nodes that participate in the distributed filesystem network.

Examples:
  dfs node start --listen :3001                    # Start a node
  dfs node start --listen :3001 --bootstrap /p2p/... # Start with bootstrap peers
  dfs node status                                   # Show node status`,
}

// nodeStartCmd represents the node start command
var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a storage node",
	Long: `Start a storage node that participates in the distributed filesystem network.

The node will:
- Listen on the specified address
- Bootstrap to the DHT
- Accept incoming connections
- Store and serve blocks
- Participate in block exchange (bitswap)

Examples:
  dfs node start --listen :3001
  dfs node start --listen :3001 --bootstrap /ip4/10.0.0.1/tcp/4001/p2p/QmPeerID`,
	RunE: runNodeStart,
}

// nodeStatusCmd represents the node status command
var nodeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show node status",
	RunE:  runNodeStatus,
}

var (
	nodeListenAddr  string
	nodeBootstrap   []string
	nodeEnableRelay bool
	nodeEnableDHT   bool
)

func init() {
	nodeCmd.AddCommand(nodeStartCmd)
	nodeCmd.AddCommand(nodeStatusCmd)

	nodeStartCmd.Flags().StringVar(&nodeListenAddr, "listen", ":3001", "address to listen on")
	nodeStartCmd.Flags().StringSliceVar(&nodeBootstrap, "bootstrap", []string{}, "bootstrap peer addresses")
	nodeStartCmd.Flags().BoolVar(&nodeEnableRelay, "relay", false, "enable relay transport")
	nodeStartCmd.Flags().BoolVar(&nodeEnableDHT, "dht", true, "enable DHT")
}

func runNodeStart(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting node on %s\n", nodeListenAddr)

	if len(nodeBootstrap) > 0 {
		fmt.Printf("Bootstrap peers: %v\n", nodeBootstrap)
	}

	// Extract port from listen address
	listenPort := 3001
	fmt.Sscanf(nodeListenAddr, ":%d", &listenPort)

	// Create libp2p node
	node, err := mcp.NewLibp2pNode(listenPort, nil)
	if err != nil {
		return fmt.Errorf("failed to create libp2p node: %w", err)
	}
	defer node.Close()

	fmt.Printf("Node ID: %s\n", node.ID())

	// Create DHT if enabled
	if nodeEnableDHT {
		dht, err := mcp.NewKadDHT(node.GetHost())
		if err != nil {
			return fmt.Errorf("failed to create DHT: %w", err)
		}
		defer dht.Close()

		// Bootstrap DHT
		if len(nodeBootstrap) > 0 {
			fmt.Println("Bootstrapping DHT...")
			if err := dht.Bootstrap(cmd.Context()); err != nil {
				fmt.Printf("Warning: DHT bootstrap failed: %v\n", err)
			}
		}

		fmt.Println("DHT enabled")
	}

	// Register protocol handler
	node.SetStreamHandler(protocol.ID("/dfs/1.0.0"), func(stream network.Stream) {
		fmt.Printf("Received stream from %s\n", stream.Conn().RemotePeer())
		stream.Close()
	})

	fmt.Println("Node started successfully")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down node...")
	return nil
}

func runNodeStatus(cmd *cobra.Command, args []string) error {
	// In a full implementation, this would connect to a running node
	// and query its status via RPC or API

	fmt.Println("Node Status:")
	fmt.Println("  Status: Not connected")
	fmt.Println("  Use 'dfs node start' to start a node")

	return nil
}
