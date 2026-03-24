package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hellodebojeet/Distribute/gateway"
	"github.com/hellodebojeet/Distribute/gateway/blockstore"
	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/hellodebojeet/Distribute/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

// gatewayCmd represents the gateway command
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the HTTP gateway",
	Long: `Start the HTTP gateway server for browser access to content.

The gateway provides:
- /ipfs/{cid} - Content retrieval by CID
- /ipns/{name} - Mutable pointer resolution
- /api/v0/* - IPFS-compatible API
- /metrics - Prometheus metrics endpoint
- /health, /ready - Health check endpoints

Examples:
  dfs gateway --listen :8080                          # Start gateway on port 8080
  dfs gateway --listen :8080 --store-dir ./data       # Custom store directory
  dfs gateway --enable-ipns                           # Enable IPNS support`,
	RunE: runGateway,
}

var (
	gatewayListenAddr   string
	gatewayStoreDir     string
	gatewayEnableIPNS   bool
	gatewayAllowOrigins []string
)

func init() {
	gatewayCmd.Flags().StringVar(&gatewayListenAddr, "listen", ":8080", "address to listen on")
	gatewayCmd.Flags().StringVar(&gatewayStoreDir, "store-dir", "./data", "storage directory")
	gatewayCmd.Flags().BoolVar(&gatewayEnableIPNS, "enable-ipns", true, "enable IPNS support")
	gatewayCmd.Flags().StringSliceVar(&gatewayAllowOrigins, "allow-origins", []string{"*"}, "allowed CORS origins")
}

func runGateway(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting HTTP gateway on %s\n", gatewayListenAddr)

	// Create storage
	store := server.NewStore(server.StoreOpts{
		Root: gatewayStoreDir,
	})

	// Create blockstore
	bs := blockstore.NewBlockStore(store)

	// Create DAG
	ca := mcp.NewIPFSContentAddresser()
	dag := mcp.NewSimpleMerkleDAG(ca)

	// Create IPNS resolver if enabled
	var ipns gateway.IPNSResolver
	if gatewayEnableIPNS {
		ipns = gateway.NewMemoryIPNSResolver()
		fmt.Println("IPNS enabled")
	}

	// Create gateway config
	config := gateway.DefaultGatewayConfig()
	config.ListenAddr = gatewayListenAddr
	config.EnableIPNS = gatewayEnableIPNS
	config.AllowedOrigins = gatewayAllowOrigins

	// Create gateway
	// Note: In a full implementation, metrics collector and logger would be properly initialized
	gw := gateway.NewGateway(config, bs, dag, ipns, nil, nil)

	// Register Prometheus metrics handler
	http.Handle("/metrics", promhttp.Handler())

	// Start gateway in goroutine
	go func() {
		if err := gw.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Gateway error: %v\n", err)
		}
	}()

	fmt.Printf("Gateway started on %s\n", gatewayListenAddr)
	fmt.Printf("  IPFS endpoint: http://%s/ipfs/{cid}\n", gatewayListenAddr)
	if gatewayEnableIPNS {
		fmt.Printf("  IPNS endpoint: http://%s/ipns/{name}\n", gatewayListenAddr)
	}
	fmt.Printf("  Metrics: http://%s/metrics\n", gatewayListenAddr)
	fmt.Printf("  Health: http://%s/health\n", gatewayListenAddr)
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down gateway...")
	return gw.Stop(cmd.Context())
}
