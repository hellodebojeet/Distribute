// Package cmd implements CLI commands for the distributed filesystem.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dfs",
	Short: "A distributed filesystem with IPFS compatibility",
	Long: `Distributed File System (dfs) - A production-ready, fault-tolerant distributed 
filesystem with IPFS compatibility using libp2p, IPLD, and modular capability providers.

Examples:
  dfs add ./myfile.txt                    # Add a file
  dfs get QmXxx ./output                  # Get a file by CID
  dfs ls QmXxx                           # List directory contents
  dfs pin add QmXxx                      # Pin content
  dfs peers list                          # List connected peers
  dfs node start --listen :3001          # Start a node
  dfs gateway --listen :8080             # Start HTTP gateway`,
}

// Global flags - these are set by persistent flags in rootCmd
var (
	CfgFile     string
	NodeAddr    string
	GatewayAddr string
	LogLevel    string
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is $HOME/.dfs.yaml)")
	rootCmd.PersistentFlags().StringVar(&NodeAddr, "node", "", "node address to connect to")
	rootCmd.PersistentFlags().StringVar(&GatewayAddr, "gateway", "http://localhost:8080", "gateway URL")
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Add subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(peersCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(versionCmd)
}

// getContext returns a context with the specified timeout
func getContext(timeout time.Duration) (cancel func()) {
	// Context handling would be implemented here
	return func() {}
}

// printJSON prints data as JSON
func printJSON(v interface{}) {
	// JSON marshaling would go here
	fmt.Printf("%+v\n", v)
}
