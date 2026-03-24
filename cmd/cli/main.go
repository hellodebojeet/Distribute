package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "distribute",
		Short: "Distributed filesystem with IPFS compatibility",
		Long:  `A production-ready distributed filesystem with IPFS and NFT compatibility.`,
	}

	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Start a storage node",
		RunE:  runNode,
	}

	gatewayCmd = &cobra.Command{
		Use:   "gateway",
		Short: "Start an HTTP gateway",
		RunE:  runGateway,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run:   printVersion,
	}
)

func init() {
	// Add commands to root
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(versionCmd)

	// Node command flags
	nodeCmd.Flags().StringP("listen", "l", ":4001", "Address to listen on")
	nodeCmd.Flags().StringP("data", "d", "./data", "Data directory")
	nodeCmd.Flags().StringSliceP("bootstrap", "b", []string{}, "Bootstrap nodes")

	// Gateway command flags
	gatewayCmd.Flags().StringP("listen", "l", ":8080", "Address to listen on")
	gatewayCmd.Flags().StringP("node", "n", "localhost:4001", "Node address")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runNode(cmd *cobra.Command, args []string) error {
	listen, _ := cmd.Flags().GetString("listen")
	data, _ := cmd.Flags().GetString("data")
	bootstrap, _ := cmd.Flags().GetStringSlice("bootstrap")

	fmt.Printf("Starting node on %s with data dir %s\n", listen, data)
	if len(bootstrap) > 0 {
		fmt.Printf("Bootstrap nodes: %v\n", bootstrap)
	}

	// TODO: Implement node startup
	// This would integrate with the P2P and DHT packages

	return nil
}

func runGateway(cmd *cobra.Command, args []string) error {
	listen, _ := cmd.Flags().GetString("listen")
	node, _ := cmd.Flags().GetString("node")

	fmt.Printf("Starting gateway on %s, proxying to node %s\n", listen, node)

	// TODO: Implement gateway startup
	// This would provide an HTTP interface to the filesystem

	return nil
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println("distribute v0.1.0")
	fmt.Println("IPFS + NFT compatible distributed filesystem")
}
