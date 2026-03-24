package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <cid> [output]",
	Short: "Retrieve content by CID",
	Long: `Retrieve content from the distributed filesystem by its CID (Content Identifier).

Examples:
  dfs get QmXxx ./output.txt
  dfs get QmXxx --output ./downloads/
  dfs get ipfs://QmXxx ./output
  dfs get QmXxx/subpath/file.txt ./output`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runGet,
}

var (
	getOutput   string
	getProgress bool
)

func init() {
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "", "output file or directory")
	getCmd.Flags().BoolVar(&getProgress, "progress", true, "show progress")
}

func runGet(cmd *cobra.Command, args []string) error {
	cidStr := args[0]

	// Handle ipfs:// URI scheme
	if len(cidStr) > 7 && cidStr[:7] == "ipfs://" {
		cidStr = cidStr[7:]
	}

	// Parse CID
	c, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	// Determine output path
	outputPath := getOutput
	if outputPath == "" && len(args) > 1 {
		outputPath = args[1]
	}
	if outputPath == "" {
		outputPath = c.String() // Use CID as filename
	}

	fmt.Printf("Retrieving %s...\n", c.String())

	// Fetch from gateway
	data, err := fetchFromGateway(c.String())
	if err != nil {
		return fmt.Errorf("failed to retrieve content: %w", err)
	}

	// Write to output
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Saved to %s (%d bytes)\n", outputPath, len(data))
	return nil
}

func fetchFromGateway(cidStr string) ([]byte, error) {
	url := fmt.Sprintf("%s/ipfs/%s", GatewayAddr, cidStr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
