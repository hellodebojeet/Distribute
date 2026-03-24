package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/common/expfmt"
	"github.com/spf13/cobra"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show system statistics",
	Long: `Display statistics about the distributed filesystem including:
- Block store usage
- Network metrics
- DAG statistics
- Replication status

Examples:
  dfs stats                      # Show all statistics
  dfs stats --format json        # Output as JSON
  dfs stats blocks               # Show block store stats only
  dfs stats network              # Show network stats only`,
	RunE: runStats,
}

var statsFormat string
var statsSection string

func init() {
	statsCmd.Flags().StringVar(&statsFormat, "format", "text", "output format: text, json, prometheus")
	statsCmd.Flags().StringVar(&statsSection, "section", "", "stats section: blocks, network, dag, replication, all")
}

type SystemStats struct {
	BlockStore  BlockStoreStats  `json:"blockstore"`
	Network     NetworkStats     `json:"network"`
	DAG         DAGStats         `json:"dag"`
	Replication ReplicationStats `json:"replication"`
	Gateway     GatewayStats     `json:"gateway"`
}

type BlockStoreStats struct {
	TotalBlocks  int64   `json:"total_blocks"`
	PinnedBlocks int64   `json:"pinned_blocks"`
	TotalSize    int64   `json:"total_size_bytes"`
	AvgBlockSize float64 `json:"avg_block_size_bytes"`
}

type NetworkStats struct {
	ConnectedPeers  int     `json:"connected_peers"`
	TotalPeers      int     `json:"total_known_peers"`
	BlocksExchanged int64   `json:"blocks_exchanged"`
	BytesSent       int64   `json:"bytes_sent"`
	BytesReceived   int64   `json:"bytes_received"`
	DHTLookupMs     float64 `json:"dht_lookup_ms_avg"`
}

type DAGStats struct {
	TotalNodes  int64 `json:"total_nodes"`
	TotalLinks  int64 `json:"total_links"`
	MaxDepth    int   `json:"max_depth"`
	OrphanNodes int64 `json:"orphan_nodes"`
}

type ReplicationStats struct {
	ReplicationFactor int     `json:"replication_factor"`
	UnderReplicated   int64   `json:"under_replicated_blocks"`
	RepairOperations  int64   `json:"repair_operations"`
	AvgReplicationMs  float64 `json:"avg_replication_ms"`
}

type GatewayStats struct {
	TotalRequests  int64   `json:"total_requests"`
	RequestsPerSec float64 `json:"requests_per_sec"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	BytesServed    int64   `json:"bytes_served"`
}

func runStats(cmd *cobra.Command, args []string) error {
	stats, err := fetchStatsFromGateway()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	switch statsFormat {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	case "prometheus":
		return printPrometheusStats(cmd)
	default:
		printStatsText(stats)
	}

	return nil
}

func fetchStatsFromGateway() (*SystemStats, error) {
	url := fmt.Sprintf("%s/stats", GatewayAddr)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats SystemStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

func printStatsText(stats *SystemStats) {
	fmt.Println("=== Distributed File System Statistics ===")
	fmt.Println()

	fmt.Println("Block Store:")
	fmt.Printf("  Total Blocks:    %d\n", stats.BlockStore.TotalBlocks)
	fmt.Printf("  Pinned Blocks:   %d\n", stats.BlockStore.PinnedBlocks)
	fmt.Printf("  Total Size:      %s\n", formatBytes(stats.BlockStore.TotalSize))
	fmt.Printf("  Avg Block Size:  %s\n", formatBytes(int64(stats.BlockStore.AvgBlockSize)))
	fmt.Println()

	fmt.Println("Network:")
	fmt.Printf("  Connected Peers: %d\n", stats.Network.ConnectedPeers)
	fmt.Printf("  Known Peers:     %d\n", stats.Network.TotalPeers)
	fmt.Printf("  Blocks Exchanged: %d\n", stats.Network.BlocksExchanged)
	fmt.Printf("  Bytes Sent:      %s\n", formatBytes(stats.Network.BytesSent))
	fmt.Printf("  Bytes Received:  %s\n", formatBytes(stats.Network.BytesReceived))
	fmt.Printf("  DHT Lookup Avg:  %.2f ms\n", stats.Network.DHTLookupMs)
	fmt.Println()

	fmt.Println("DAG:")
	fmt.Printf("  Total Nodes:     %d\n", stats.DAG.TotalNodes)
	fmt.Printf("  Total Links:     %d\n", stats.DAG.TotalLinks)
	fmt.Printf("  Max Depth:       %d\n", stats.DAG.MaxDepth)
	fmt.Printf("  Orphan Nodes:    %d\n", stats.DAG.OrphanNodes)
	fmt.Println()

	fmt.Println("Replication:")
	fmt.Printf("  Replication Factor: %d\n", stats.Replication.ReplicationFactor)
	fmt.Printf("  Under-Replicated:   %d blocks\n", stats.Replication.UnderReplicated)
	fmt.Printf("  Repair Operations:  %d\n", stats.Replication.RepairOperations)
	fmt.Println()

	fmt.Println("Gateway:")
	fmt.Printf("  Total Requests:  %d\n", stats.Gateway.TotalRequests)
	fmt.Printf("  Requests/sec:    %.2f\n", stats.Gateway.RequestsPerSec)
	fmt.Printf("  Avg Latency:     %.2f ms\n", stats.Gateway.AvgLatencyMs)
	fmt.Printf("  Bytes Served:    %s\n", formatBytes(stats.Gateway.BytesServed))
}

func printPrometheusStats(cmd *cobra.Command) error {
	// Fetch Prometheus metrics from the gateway
	url := fmt.Sprintf("%s/metrics", GatewayAddr)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse and output Prometheus format
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return err
	}

	// Write metrics in Prometheus text format
	encoder := expfmt.NewEncoder(cmd.OutOrStdout(), expfmt.FmtText)
	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			return err
		}
	}

	return nil
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
