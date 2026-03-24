package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
)

// pinCmd represents the pin command
var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: "Manage pinned content",
	Long: `Manage content pinning to ensure data persistence.

Pinning marks content to be kept locally and replicated across the network.
Unpinned content may be garbage collected.

Examples:
  dfs pin add QmXxx              # Pin content
  dfs pin rm QmXxx               # Unpin content
  dfs pin ls                     # List all pinned content`,
}

// pinAddCmd represents the pin add command
var pinAddCmd = &cobra.Command{
	Use:   "add <cid> [cids...]",
	Short: "Pin content by CID",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPinAdd,
}

// pinRmCmd represents the pin rm command
var pinRmCmd = &cobra.Command{
	Use:   "rm <cid> [cids...]",
	Short: "Unpin content by CID",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPinRm,
}

// pinLsCmd represents the pin ls command
var pinLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List pinned content",
	RunE:  runPinLs,
}

var pinLsType string
var pinLsQuiet bool

func init() {
	pinCmd.AddCommand(pinAddCmd)
	pinCmd.AddCommand(pinRmCmd)
	pinCmd.AddCommand(pinLsCmd)

	pinLsCmd.Flags().StringVar(&pinLsType, "type", "all", "pin type: all, direct, recursive, indirect")
	pinLsCmd.Flags().BoolVarP(&pinLsQuiet, "quiet", "q", false, "print CIDs only")
}

func runPinAdd(cmd *cobra.Command, args []string) error {
	for _, cidStr := range args {
		c, err := cid.Decode(cidStr)
		if err != nil {
			return fmt.Errorf("invalid CID %s: %w", cidStr, err)
		}

		if err := pinViaGateway(c.String()); err != nil {
			return fmt.Errorf("failed to pin %s: %w", cidStr, err)
		}

		if !pinLsQuiet {
			fmt.Printf("pinned %s\n", c.String())
		}
	}
	return nil
}

func runPinRm(cmd *cobra.Command, args []string) error {
	for _, cidStr := range args {
		c, err := cid.Decode(cidStr)
		if err != nil {
			return fmt.Errorf("invalid CID %s: %w", cidStr, err)
		}

		if err := unpinViaGateway(c.String()); err != nil {
			return fmt.Errorf("failed to unpin %s: %w", cidStr, err)
		}

		if !pinLsQuiet {
			fmt.Printf("unpinned %s\n", c.String())
		}
	}
	return nil
}

func runPinLs(cmd *cobra.Command, args []string) error {
	pins, err := listPinsViaGateway()
	if err != nil {
		return fmt.Errorf("failed to list pins: %w", err)
	}

	if pinLsQuiet {
		for cid := range pins {
			fmt.Println(cid)
		}
	} else {
		fmt.Printf("%-64s %s\n", "CID", "TYPE")
		fmt.Printf("%-64s %s\n", "---", "----")
		for cid, pinType := range pins {
			fmt.Printf("%-64s %s\n", cid, pinType)
		}
	}

	return nil
}

func pinViaGateway(cidStr string) error {
	url := fmt.Sprintf("%s/api/v0/pin/add/%s", GatewayAddr, cidStr)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}
	return nil
}

func unpinViaGateway(cidStr string) error {
	url := fmt.Sprintf("%s/api/v0/pin/rm/%s", GatewayAddr, cidStr)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}
	return nil
}

func listPinsViaGateway() (map[string]string, error) {
	url := fmt.Sprintf("%s/api/v0/pin/ls", GatewayAddr)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	var result struct {
		Keys map[string]struct {
			Type string `json:"Type"`
		} `json:"Keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	pins := make(map[string]string)
	for cid, info := range result.Keys {
		pins[cid] = strings.ToLower(info.Type)
	}

	return pins, nil
}
