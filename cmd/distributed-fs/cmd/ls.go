package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls <cid>",
	Short: "List directory contents",
	Long: `List the contents of a directory or file links at a given CID.

Examples:
  dfs ls QmXxx                                    # List root directory
  dfs ls QmXxx --format json                      # Output as JSON
  dfs ls ipfs://QmXxx/subdir                      # List subdirectory`,
	Args: cobra.ExactArgs(1),
	RunE: runLs,
}

var lsFormat string

func init() {
	lsCmd.Flags().StringVar(&lsFormat, "format", "table", "output format: table, json")
}

func runLs(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Listing %s...\n", c.String())

	// Fetch directory listing from gateway
	entries, err := fetchDirectoryListing(c.String())
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}

	switch lsFormat {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	default:
		printDirectoryTable(entries)
	}

	return nil
}

type DirectoryEntry struct {
	Name   string `json:"Name"`
	CID    string `json:"CID"`
	Size   int64  `json:"Size,omitempty"`
	Type   string `json:"Type"`
	Target string `json:"Target,omitempty"`
}

func fetchDirectoryListing(cidStr string) ([]DirectoryEntry, error) {
	url := fmt.Sprintf("%s/api/v0/ls?arg=%s", GatewayAddr, cidStr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	var result struct {
		Objects []struct {
			Hash  string           `json:"Hash"`
			Links []DirectoryEntry `json:"Links"`
		} `json:"Objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Objects) > 0 {
		return result.Objects[0].Links, nil
	}

	return []DirectoryEntry{}, nil
}

func printDirectoryTable(entries []DirectoryEntry) {
	if len(entries) == 0 {
		fmt.Println("empty directory")
		return
	}

	fmt.Printf("%-50s %-10s %s\n", "NAME", "SIZE", "CID")
	fmt.Printf("%-50s %-10s %s\n", "----", "----", "---")

	for _, e := range entries {
		sizeStr := formatSize(e.Size)
		if e.Type == "dir" {
			sizeStr = "-"
		}
		fmt.Printf("%-50s %-10s %s\n", e.Name, sizeStr, e.CID)
	}
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
