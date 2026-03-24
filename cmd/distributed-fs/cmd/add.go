package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [files...]",
	Short: "Add files to the distributed filesystem",
	Long: `Add files to the distributed filesystem by chunking, hashing, and storing.

Examples:
  dfs add ./myfile.txt                           # Add a single file
  dfs add ./folder/                              # Add a directory recursively
  dfs add file1.txt file2.txt                    # Add multiple files
  dfs add --chunk-size 262144 ./largefile.bin    # Custom chunk size
  dfs add --pin ./important.txt                  # Add and pin the file`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

var (
	addChunkSize int
	addPin       bool
	addRawLeaves bool
)

func init() {
	addCmd.Flags().IntVar(&addChunkSize, "chunk-size", 262144, "chunk size in bytes (default 256KB)")
	addCmd.Flags().BoolVar(&addPin, "pin", false, "pin the added content")
	addCmd.Flags().BoolVar(&addRawLeaves, "raw-leaves", true, "use raw leaves for leaf nodes")
}

func runAdd(cmd *cobra.Command, args []string) error {
	fmt.Printf("Adding %d file(s)...\n", len(args))

	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() {
			if err := addDirectory(path); err != nil {
				return err
			}
		} else {
			cid, err := addFile(path)
			if err != nil {
				return err
			}
			fmt.Printf("added %s -> %s\n", path, cid)
		}
	}

	return nil
}

func addFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// In a full implementation, this would:
	// 1. Chunk the data using the specified chunk size
	// 2. Hash each chunk to get CID
	// 3. Build the Merkle DAG
	// 4. Store blocks via blockstore
	// 5. Provide CID to DHT
	// 6. Return root CID

	// For now, use gateway API if available
	if GatewayAddr != "" {
		return addToGateway(path, f)
	}

	return "", fmt.Errorf("no gateway specified, use --gateway flag")
}

func addToGateway(path string, r io.Reader) (string, error) {
	// POST /api/v0/add
	// This would use multipart form to upload file to gateway API
	return "", fmt.Errorf("gateway upload not yet implemented")
}

func addDirectory(path string) error {
	fmt.Printf("Adding directory: %s\n", path)
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			cid, err := addFile(p)
			if err != nil {
				return err
			}
			fmt.Printf("  %s -> %s\n", p, cid)
		}
		return nil
	})
}
