package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "distributed-fs",
	Short: "A distributed filesystem with IPFS compatibility",
	Long:  `A production-ready, fault-tolerant distributed filesystem with IPFS compatibility using libp2p, IPLD, and other modular capability providers.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.distributed-fs.yaml)")
	rootCmd.PersistentFlags().BoolP("toggle", "t", false, "Help message for toggle")

	// Add subcommands
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(versionCmd)
}

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run a storage node",
	Long:  `Start a storage node that participates in the distributed filesystem network.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		listenAddr, _ := cmd.Flags().GetString("listen")
		bootstrapNodes, _ := cmd.Flags().GetStringSlice("bootstrap")

		// Create and start the node
		startNode(listenAddr, bootstrapNodes)
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of distributed-fs",
	Long:  `All software has versions. This is distributed-fs's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("distributed-fs v0.1.0 -- HEAD")
	},
}

func init() {
	// Node command flags
	nodeCmd.Flags().StringP("listen", "l", ":3001", "Address to listen on")
	nodeCmd.Flags().StringSliceP("bootstrap", "b", []string{}, "Bootstrap nodes to connect to")
}

func startNode(listenAddr string, bootstrapNodes []string) {
	fmt.Printf("Starting node on %s\n", listenAddr)
	if len(bootstrapNodes) > 0 {
		fmt.Printf("Bootstrap nodes: %v\n", bootstrapNodes)
	}

	// Create libp2p node
	listenPort := 3001
	if listenAddr != "" {
		fmt.Sscanf(listenAddr, ":%d", &listenPort)
	}

	node, err := mcp.NewLibp2pNode(listenPort, nil) // Pass nil for now to generate a key
	if err != nil {
		fmt.Printf("Failed to create libp2p node: %v\n", err)
		return
	}
	defer node.Close()

	fmt.Printf("Node ID: %s\n", node.ID())

	// Create DHT for peer discovery
	dht, err := mcp.NewKadDHT(node.GetHost())
	if err != nil {
		fmt.Printf("Failed to create DHT: %v\n", err)
		return
	}
	defer dht.Close()

	// Bootstrap the DHT if we have bootstrap nodes
	if len(bootstrapNodes) > 0 {
		ctx := context.Background()
		for _, bootstrapAddr := range bootstrapNodes {
			// Parse bootstrap address and add to DHT
			// This is simplified - in practice we'd properly parse multiaddrs
			fmt.Printf("Bootstrapping with %s\n", bootstrapAddr)
		}

		// Actual bootstrap
		if err := dht.Bootstrap(ctx); err != nil {
			fmt.Printf("Failed to bootstrap DHT: %v\n", err)
		}
	}

	// Set up protocol handlers using the P2PNode interface
	node.RegisterHandler("/distributed-fs/1.0.0", func(from string, msg []byte) error {
		// Handle incoming message
		fmt.Printf("Received message from %s: %s\n", from, string(msg))
		return nil
	})

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nShutting down node...")
}
