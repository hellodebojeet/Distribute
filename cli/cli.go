// Package cli provides command-line interface commands for the distributed filesystem.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"

	"github.com/hellodebojeet/Distribute/gateway/blockstore"
	"github.com/hellodebojeet/Distribute/internal/mcp"
)

// CLI holds the CLI dependencies.
type CLI struct {
	RootCmd       *cobra.Command
	BlockStore    *blockstore.BlockStore
	ContentAddr   mcp.ContentAddresser
	DHT           mcp.DHT
	BlockExchange mcp.BlockExchanger
	NodeID        string
}

// NewCLI creates a new CLI instance.
func NewCLI(nodeID string, bs *blockstore.BlockStore, dht mcp.DHT, be mcp.BlockExchanger) *CLI {
	cli := &CLI{
		NodeID:        nodeID,
		BlockStore:    bs,
		ContentAddr:   mcp.NewIPFSContentAddresser(),
		DHT:           dht,
		BlockExchange: be,
	}

	cli.RootCmd = &cobra.Command{
		Use:   "dfs",
		Short: "Distributed Filesystem CLI",
		Long:  `A command-line interface for the distributed content-addressed filesystem.`,
	}

	cli.setupCommands()
	return cli
}

// setupCommands registers all CLI commands.
func (cli *CLI) setupCommands() {
	cli.RootCmd.AddCommand(cli.addCmd())
	cli.RootCmd.AddCommand(cli.getCmd())
	cli.RootCmd.AddCommand(cli.peersCmd())
	cli.RootCmd.AddCommand(cli.statsCmd())
	cli.RootCmd.AddCommand(cli.pinCmd())
	cli.RootCmd.AddCommand(cli.unpinCmd())
	cli.RootCmd.AddCommand(cli.lsPinnedCmd())
	cli.RootCmd.AddCommand(cli.findProvsCmd())
	cli.RootCmd.AddCommand(cli.catCmd())
}

// addCmd returns the 'add' command.
func (cli *CLI) addCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [file]",
		Short: "Add a file to the local node",
		Long:  `Add a file to the local node, computing its CID and storing it.`,
		Args:  cobra.ExactArgs(1),
		RunE:  cli.runAdd,
	}
}

func (cli *CLI) runAdd(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Compute CID
	fileCID := cli.ContentAddr.Hash(data)

	// Store the block
	if err := cli.BlockStore.PutBlock(context.Background(), fileCID, data); err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}

	// Announce to DHT
	if cli.DHT != nil {
		ctx := context.Background()
		if err := cli.DHT.Provide(ctx, fileCID, true); err != nil {
			// Log but don't fail - content is stored locally
			fmt.Fprintf(os.Stderr, "warning: failed to announce to DHT: %v\n", err)
		}
	}

	// Print result
	fmt.Printf("added %s %s\n", fileCID.String(), filepath.Base(filePath))
	return nil
}

// getCmd returns the 'get' command.
func (cli *CLI) getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [cid] [output]",
		Short: "Get content by CID",
		Long:  `Retrieve content by CID and save to a file.`,
		Args:  cobra.ExactArgs(2),
		RunE:  cli.runGet,
	}
}

func (cli *CLI) runGet(cmd *cobra.Command, args []string) error {
	cidStr := args[0]
	outputPath := args[1]

	// Parse CID
	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	// Try local blockstore first
	data, err := cli.BlockStore.GetBlock(context.Background(), parsedCID)
	if err != nil {
		// Try block exchange (network)
		data, err = cli.BlockExchange.GetBlock(context.Background(), parsedCID)
		if err != nil {
			return fmt.Errorf("failed to get block: %w", err)
		}
	}

	// Verify hash
	computedCID := cli.ContentAddr.Hash(data)
	if !computedCID.Equals(parsedCID) {
		return fmt.Errorf("hash mismatch: expected %s, got %s", parsedCID, computedCID)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("saved %s to %s\n", cidStr, outputPath)
	return nil
}

// catCmd returns the 'cat' command (output to stdout).
func (cli *CLI) catCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat [cid]",
		Short: "Output content by CID to stdout",
		Long:  `Retrieve content by CID and write to stdout.`,
		Args:  cobra.ExactArgs(1),
		RunE:  cli.runCat,
	}
}

func (cli *CLI) runCat(cmd *cobra.Command, args []string) error {
	cidStr := args[0]

	// Parse CID
	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	// Try local blockstore first
	data, err := cli.BlockStore.GetBlock(context.Background(), parsedCID)
	if err != nil {
		// Try block exchange (network)
		data, err = cli.BlockExchange.GetBlock(context.Background(), parsedCID)
		if err != nil {
			return fmt.Errorf("failed to get block: %w", err)
		}
	}

	// Verify hash
	computedCID := cli.ContentAddr.Hash(data)
	if !computedCID.Equals(parsedCID) {
		return fmt.Errorf("hash mismatch: expected %s, got %s", parsedCID, computedCID)
	}

	// Write to stdout
	_, err = os.Stdout.Write(data)
	return err
}

// peersCmd returns the 'peers' command.
func (cli *CLI) peersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "peers",
		Short: "List connected peers",
		Long:  `Display a list of peers connected to this node.`,
		RunE:  cli.runPeers,
	}
}

func (cli *CLI) runPeers(cmd *cobra.Command, args []string) error {
	fmt.Printf("Node ID: %s\n", cli.NodeID)
	fmt.Println("\nConnected peers:")
	fmt.Println("  (DHT-based peer discovery - see logs for peer connections)")
	return nil
}

// statsCmd returns the 'stats' command.
func (cli *CLI) statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show node statistics",
		Long:  `Display statistics about the local node.`,
		RunE:  cli.runStats,
	}
}

func (cli *CLI) runStats(cmd *cobra.Command, args []string) error {
	// Get pinned count
	pinned := cli.BlockStore.ListPinned()

	fmt.Println("Node Statistics:")
	fmt.Println("================")
	fmt.Printf("Node ID:      %s\n", cli.NodeID)
	fmt.Printf("Pinned CIDs:  %d\n", len(pinned))

	if len(pinned) > 0 {
		fmt.Println("\nPinned Content:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CID\t")
		for _, c := range pinned {
			fmt.Fprintf(w, "%s\t\n", c.String())
		}
		w.Flush()
	}

	return nil
}

// pinCmd returns the 'pin' command.
func (cli *CLI) pinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin [cid]",
		Short: "Pin content by CID",
		Long:  `Pin content to ensure it is not garbage collected.`,
		Args:  cobra.ExactArgs(1),
		RunE:  cli.runPin,
	}
}

func (cli *CLI) runPin(cmd *cobra.Command, args []string) error {
	cidStr := args[0]

	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	if err := cli.BlockStore.Pin(parsedCID); err != nil {
		return fmt.Errorf("failed to pin: %w", err)
	}

	fmt.Printf("pinned %s\n", cidStr)
	return nil
}

// unpinCmd returns the 'unpin' command.
func (cli *CLI) unpinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin [cid]",
		Short: "Unpin content by CID",
		Long:  `Remove pin from content, allowing it to be garbage collected.`,
		Args:  cobra.ExactArgs(1),
		RunE:  cli.runUnpin,
	}
}

func (cli *CLI) runUnpin(cmd *cobra.Command, args []string) error {
	cidStr := args[0]

	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	if err := cli.BlockStore.Unpin(parsedCID); err != nil {
		return fmt.Errorf("failed to unpin: %w", err)
	}

	fmt.Printf("unpinned %s\n", cidStr)
	return nil
}

// lsPinnedCmd returns the 'ls-pinned' command.
func (cli *CLI) lsPinnedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls-pinned",
		Short: "List all pinned CIDs",
		Long:  `List all CIDs that are pinned locally.`,
		RunE:  cli.runLsPinned,
	}
}

func (cli *CLI) runLsPinned(cmd *cobra.Command, args []string) error {
	pinned := cli.BlockStore.ListPinned()

	if len(pinned) == 0 {
		fmt.Println("No pinned content")
		return nil
	}

	fmt.Printf("Pinned content (%d items):\n", len(pinned))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CID\tPinned")
	for _, c := range pinned {
		fmt.Fprintf(w, "%s\tyes\t\n", c.String())
	}
	w.Flush()

	return nil
}

// findProvsCmd returns the 'findprovs' command.
func (cli *CLI) findProvsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "findprovs [cid]",
		Short: "Find providers for a CID",
		Long:  `Search the DHT for peers that provide content with the given CID.`,
		Args:  cobra.ExactArgs(1),
		RunE:  cli.runFindProvs,
	}
}

func (cli *CLI) runFindProvs(cmd *cobra.Command, args []string) error {
	cidStr := args[0]

	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	if cli.DHT == nil {
		return fmt.Errorf("DHT not available")
	}

	fmt.Printf("Searching for providers of %s...\n", cidStr)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	providers := cli.DHT.FindProvidersAsync(ctx, parsedCID, 20)

	count := 0
	for provider := range providers {
		count++
		fmt.Printf("provider %d: %s\n", count, provider.ID.String())
	}

	if count == 0 {
		fmt.Println("no providers found")
	}

	return nil
}

// Execute runs the CLI.
func (cli *CLI) Execute() error {
	return cli.RootCmd.Execute()
}
