package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

// peersCmd represents the peers command
var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "Manage connected peers",
	Long: `View and manage connected peers in the P2P network.

Examples:
  dfs peers list                # List connected peers
  dfs peers connect <peer>     # Connect to a peer
  dfs peers stat                # Show peer statistics`,
}

// peersListCmd represents the peers list command
var peersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected peers",
	RunE:  runPeersList,
}

// peersConnectCmd represents the peers connect command
var peersConnectCmd = &cobra.Command{
	Use:   "connect <peer-address>",
	Short: "Connect to a peer",
	Args:  cobra.ExactArgs(1),
	RunE:  runPeersConnect,
}

// peersStatCmd represents the peers stat command
var peersStatCmd = &cobra.Command{
	Use:   "stat",
	Short: "Show peer statistics",
	RunE:  runPeersStat,
}

var peersFormat string

func init() {
	peersCmd.AddCommand(peersListCmd)
	peersCmd.AddCommand(peersConnectCmd)
	peersCmd.AddCommand(peersStatCmd)

	peersListCmd.Flags().StringVar(&peersFormat, "format", "table", "output format: table, json")
}

type PeerInfo struct {
	ID        string   `json:"ID"`
	Addresses []string `json:"Addresses"`
	Connected bool     `json:"Connected"`
	Latency   string   `json:"Latency,omitempty"`
	Streams   []string `json:"Streams,omitempty"`
}

func runPeersList(cmd *cobra.Command, args []string) error {
	peers, err := fetchPeersFromGateway()
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	switch peersFormat {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(peers)
	default:
		printPeersTable(peers)
	}

	return nil
}

func runPeersConnect(cmd *cobra.Command, args []string) error {
	peerAddr := args[0]
	fmt.Printf("Connecting to %s...\n", peerAddr)

	// In a full implementation, this would use the P2P layer to connect
	// For now, return a placeholder
	fmt.Printf("Connect functionality requires a running node\n")
	return nil
}

func runPeersStat(cmd *cobra.Command, args []string) error {
	stats, err := fetchPeerStatsFromGateway()
	if err != nil {
		return fmt.Errorf("failed to get peer stats: %w", err)
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(stats)
}

func fetchPeersFromGateway() ([]PeerInfo, error) {
	url := fmt.Sprintf("%s/api/v0/swarm/peers", GatewayAddr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	var result struct {
		Peers []PeerInfo `json:"Peers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Peers, nil
}

func fetchPeerStatsFromGateway() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v0/swarm/stat", GatewayAddr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return stats, nil
}

func printPeersTable(peers []PeerInfo) {
	if len(peers) == 0 {
		fmt.Println("No connected peers")
		return
	}

	fmt.Printf("%-50s %-15s %s\n", "PEER ID", "STATUS", "LATENCY")
	fmt.Printf("%-50s %-15s %s\n", "-------", "------", "-------")

	for _, peer := range peers {
		status := "disconnected"
		if peer.Connected {
			status = "connected"
		}
		latency := peer.Latency
		if latency == "" {
			latency = "-"
		}
		fmt.Printf("%-50s %-15s %s\n", peer.ID, status, latency)
	}
}
