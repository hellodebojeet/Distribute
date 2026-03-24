// Package gateway provides HTTP access to content-addressed data with DAG traversal.
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/hellodebojeet/Distribute/gateway/blockstore"
	"github.com/hellodebojeet/Distribute/internal/mcp"
	"github.com/hellodebojeet/Distribute/observability/logging"
	"github.com/hellodebojeet/Distribute/observability/metrics"
	"github.com/ipfs/go-cid"
)

// GatewayConfig holds configuration for the HTTP gateway.
type GatewayConfig struct {
	ListenAddr             string
	AllowedOrigins         []string
	MaxCacheSize           int64
	CacheTTL               time.Duration
	EnableDirectoryListing bool
	EnableIPNS             bool
	MaxDAGDepth            int
}

// DefaultGatewayConfig returns sensible defaults for the gateway.
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		ListenAddr:             ":8080",
		AllowedOrigins:         []string{"*"},
		MaxCacheSize:           1 << 30, // 1GB
		CacheTTL:               5 * time.Minute,
		EnableDirectoryListing: true,
		EnableIPNS:             true,
		MaxDAGDepth:            32,
	}
}

// Gateway provides HTTP access to content-addressed data.
type Gateway struct {
	config         GatewayConfig
	blockExchanger mcp.BlockExchanger
	blockStore     *blockstore.BlockStore
	dag            mcp.MerkleDAG
	ipns           IPNSResolver
	metrics        *metrics.GatewayMetrics
	router         *mux.Router
	server         *http.Server
	logger         *logging.Logger
}

// NewGateway creates a new HTTP gateway.
func NewGateway(
	config GatewayConfig,
	bs *blockstore.BlockStore,
	dag mcp.MerkleDAG,
	ipns IPNSResolver,
	metricsCollector *metrics.Collector,
	logger *logging.Logger,
) *Gateway {
	// Handle nil logger
	if logger == nil {
		logger, _ = logging.NewLogger(logging.LevelInfo, false)
	}

	// Handle nil metrics
	var gatewayMetrics *metrics.GatewayMetrics
	if metricsCollector != nil {
		gatewayMetrics = metricsCollector.Gateway
	}

	g := &Gateway{
		config:         config,
		blockStore:     bs,
		blockExchanger: blockstore.NewBlockExchangerAdapter(bs),
		dag:            dag,
		ipns:           ipns,
		metrics:        gatewayMetrics,
		logger:         logger.With(logging.String("component", "gateway")),
		router:         mux.NewRouter(),
	}

	g.setupRoutes()
	return g
}

// setupRoutes configures the HTTP routes.
func (g *Gateway) setupRoutes() {
	// Main IPFS path handlers
	g.router.HandleFunc("/ipfs/{cid}", g.handleGetByCID).Methods("GET", "HEAD")
	g.router.HandleFunc("/ipfs/{cid}/{path:.*}", g.handleGetByCIDPath).Methods("GET", "HEAD")

	// IPNS handlers
	if g.config.EnableIPNS {
		g.router.HandleFunc("/ipns/{name}", g.handleGetByIPNS).Methods("GET", "HEAD")
		g.router.HandleFunc("/ipns/{name}/{path:.*}", g.handleGetByIPNSPath).Methods("GET", "HEAD")
	}

	// API endpoints
	g.router.HandleFunc("/api/v0/add", g.handleAdd).Methods("POST")
	g.router.HandleFunc("/api/v0/cat", g.handleCat).Methods("POST")
	g.router.HandleFunc("/api/v0/ls", g.handleLs).Methods("GET", "POST")
	g.router.HandleFunc("/api/v0/pin/add/{cid}", g.handlePinAdd).Methods("POST")
	g.router.HandleFunc("/api/v0/pin/rm/{cid}", g.handlePinRm).Methods("POST")
	g.router.HandleFunc("/api/v0/pin/ls", g.handlePinList).Methods("GET", "POST")

	// Health and metrics endpoints
	g.router.HandleFunc("/health", g.handleHealth).Methods("GET")
	g.router.HandleFunc("/ready", g.handleReady).Methods("GET")
	g.router.HandleFunc("/stats", g.handleStats).Methods("GET")
	g.router.HandleFunc("/metrics", g.handlePrometheusMetrics).Methods("GET")

	// CORS middleware
	g.router.Use(g.corsMiddleware)
	g.router.Use(g.metricsMiddleware)
}

// Start begins serving HTTP requests.
func (g *Gateway) Start() error {
	g.server = &http.Server{
		Addr:         g.config.ListenAddr,
		Handler:      g.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	g.logger.Info("gateway started", logging.String("addr", g.config.ListenAddr))
	return g.server.ListenAndServe()
}

// Stop gracefully shuts down the gateway.
func (g *Gateway) Stop(ctx context.Context) error {
	if g.server != nil {
		return g.server.Shutdown(ctx)
	}
	return nil
}

// handleGetByCID serves content by CID.
func (g *Gateway) handleGetByCID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cidStr := vars["cid"]

	c, err := cid.Decode(cidStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid CID: %v", err), http.StatusBadRequest)
		return
	}

	g.serveContent(w, r, c, "")
}

// handleGetByCIDPath serves content by CID with a subpath.
func (g *Gateway) handleGetByCIDPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cidStr := vars["cid"]
	path := vars["path"]

	c, err := cid.Decode(cidStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid CID: %v", err), http.StatusBadRequest)
		return
	}

	g.serveContent(w, r, c, path)
}

// handleGetByIPNS serves content by IPNS name.
func (g *Gateway) handleGetByIPNS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if g.ipns == nil {
		http.Error(w, "IPNS not enabled", http.StatusNotImplemented)
		return
	}

	// Resolve IPNS name to CID
	resolvedCID, err := g.ipns.Resolve(r.Context(), name)
	if err != nil {
		http.Error(w, fmt.Sprintf("IPNS resolution failed: %v", err), http.StatusNotFound)
		return
	}

	g.serveContent(w, r, resolvedCID, "")
}

// handleGetByIPNSPath serves content by IPNS name with subpath.
func (g *Gateway) handleGetByIPNSPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	path := vars["path"]

	if g.ipns == nil {
		http.Error(w, "IPNS not enabled", http.StatusNotImplemented)
		return
	}

	// Resolve IPNS name to CID
	resolvedCID, err := g.ipns.Resolve(r.Context(), name)
	if err != nil {
		http.Error(w, fmt.Sprintf("IPNS resolution failed: %v", err), http.StatusNotFound)
		return
	}

	g.serveContent(w, r, resolvedCID, path)
}

// serveContent retrieves and serves content for the given CID.
func (g *Gateway) serveContent(w http.ResponseWriter, r *http.Request, c cid.Cid, subpath string) {
	start := time.Now()
	ctx := r.Context()

	// Try to get block data directly first
	data, err := g.blockExchanger.GetBlock(ctx, c)
	if err != nil {
		// Block not found locally, try to get from DAG
		data, err = g.dag.GetNode(ctx, c)
		if err != nil {
			g.metrics.RequestsTotal.WithLabelValues(r.Method, "404", "/ipfs").Inc()
			http.Error(w, fmt.Sprintf("block not found: %v", err), http.StatusNotFound)
			return
		}
	}

	// Verify the hash matches the CID
	if !verifyBlockHash(c, data) {
		g.metrics.RequestsTotal.WithLabelValues(r.Method, "422", "/ipfs").Inc()
		http.Error(w, "hash mismatch: block data corrupted", http.StatusUnprocessableEntity)
		return
	}

	// Handle subpath for directory-like content
	if subpath != "" {
		if err := g.serveSubpath(w, r, c, data, subpath); err != nil {
			g.metrics.RequestsTotal.WithLabelValues(r.Method, "404", "/ipfs").Inc()
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}

	// Check for range request
	if g.handleRangeRequest(w, r, data, c) {
		return
	}

	// Set appropriate headers
	w.Header().Set("X-CID", c.String())
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(g.config.CacheTTL.Seconds())))
	w.Header().Set("Accept-Ranges", "bytes")

	// Detect content type
	contentType := detectContentType(data)
	w.Header().Set("Content-Type", contentType)

	// Stream the content
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)

	if r.Method != "HEAD" {
		w.Write(data)
	}

	// Update metrics
	g.metrics.BytesServed.Add(float64(len(data)))
	g.metrics.RequestsTotal.WithLabelValues(r.Method, "200", "/ipfs").Inc()
	g.logger.Debug("served content",
		logging.CID(c.String()),
		logging.Int("size", len(data)),
		logging.Duration("duration", time.Since(start)),
	)
}

// serveSubpath handles paths within a DAG node using DAG traversal.
func (g *Gateway) serveSubpath(w http.ResponseWriter, r *http.Request, rootCID cid.Cid, rootData []byte, subpath string) error {
	start := time.Now()
	ctx := r.Context()

	// Parse subpath into segments
	segments := strings.Split(subpath, "/")

	// Traverse the DAG
	currentCID := rootCID
	currentData := rootData
	depth := 0

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		if depth > g.config.MaxDAGDepth {
			return fmt.Errorf("DAG depth exceeded maximum of %d", g.config.MaxDAGDepth)
		}

		// Get links from current node
		links, err := g.dag.GetLinks(ctx, currentCID)
		if err != nil {
			return fmt.Errorf("failed to get links for CID %s: %v", currentCID, err)
		}

		// Find the link matching the segment
		found := false
		for _, link := range links {
			if link.Name == segment {
				// Found the link, get the target node
				targetData, err := g.dag.GetNode(ctx, link.CID)
				if err != nil {
					// Try block store
					targetData, err = g.blockExchanger.GetBlock(ctx, link.CID)
					if err != nil {
						return fmt.Errorf("failed to get target node %s: %v", link.CID, err)
					}
				}

				currentCID = link.CID
				currentData = targetData
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("path segment %q not found", segment)
		}

		depth++
	}

	// Record DAG traversal metrics
	g.metrics.RequestDuration.WithLabelValues(r.Method, "/ipfs").Observe(time.Since(start).Seconds())

	// Serve the resolved content
	contentType := detectContentType(currentData)
	w.Header().Set("X-CID", currentCID.String())
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(currentData)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)

	if r.Method != "HEAD" {
		w.Write(currentData)
	}

	return nil
}

// handleRangeRequest handles HTTP range requests.
func (g *Gateway) handleRangeRequest(w http.ResponseWriter, r *http.Request, data []byte, c cid.Cid) bool {
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		return false
	}

	// Parse range header (e.g., "bytes=0-1023")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		http.Error(w, "invalid range header", http.StatusBadRequest)
		return true
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeSpec, "-", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid range header", http.StatusBadRequest)
		return true
	}

	totalSize := int64(len(data))
	var start, end int64

	// Parse start
	if parts[0] != "" {
		var err error
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 {
			http.Error(w, "invalid range header", http.StatusBadRequest)
			return true
		}
	}

	// Parse end
	if parts[1] != "" {
		var err error
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < 0 {
			http.Error(w, "invalid range header", http.StatusBadRequest)
			return true
		}
	} else {
		end = totalSize - 1
	}

	// Validate range
	if start >= totalSize {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return true
	}

	if end >= totalSize {
		end = totalSize - 1
	}

	// Calculate range length
	length := end - start + 1

	// Set headers for partial content
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
	w.Header().Set("Content-Type", detectContentType(data))
	w.Header().Set("X-CID", c.String())
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)

	if r.Method != "HEAD" {
		w.Write(data[start : end+1])
	}

	g.metrics.BytesServed.Add(float64(length))
	g.metrics.RequestsTotal.WithLabelValues(r.Method, "206", "/ipfs").Inc()

	return true
}

// handleAdd handles file upload via API.
func (g *Gateway) handleAdd(w http.ResponseWriter, r *http.Request) {
	// Implementation for adding files
	http.Error(w, "add not yet implemented", http.StatusNotImplemented)
}

// handleCat handles content retrieval via API.
func (g *Gateway) handleCat(w http.ResponseWriter, r *http.Request) {
	// Implementation for cat command
	http.Error(w, "cat not yet implemented", http.StatusNotImplemented)
}

// handleLs handles directory listing via API.
func (g *Gateway) handleLs(w http.ResponseWriter, r *http.Request) {
	// Implementation for ls command
	http.Error(w, "ls not yet implemented", http.StatusNotImplemented)
}

// handlePinAdd handles pin add via API.
func (g *Gateway) handlePinAdd(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cidStr := vars["cid"]

	c, err := cid.Decode(cidStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid CID: %v", err), http.StatusBadRequest)
		return
	}

	if err := g.blockStore.Pin(c); err != nil {
		http.Error(w, fmt.Sprintf("failed to pin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"Pinned":"%s"}`, c.String())
}

// handlePinRm handles pin remove via API.
func (g *Gateway) handlePinRm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cidStr := vars["cid"]

	c, err := cid.Decode(cidStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid CID: %v", err), http.StatusBadRequest)
		return
	}

	if err := g.blockStore.Unpin(c); err != nil {
		http.Error(w, fmt.Sprintf("failed to unpin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"Unpinned":"%s"}`, c.String())
}

// handlePinList handles pin listing via API.
func (g *Gateway) handlePinList(w http.ResponseWriter, r *http.Request) {
	pinned := g.blockStore.ListPinned()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Keys":{`))

	for i, c := range pinned {
		if i > 0 {
			w.Write([]byte(","))
		}
		fmt.Fprintf(w, `"%s":{"Type":"recursive"}`, c.String())
	}

	w.Write([]byte(`}}`))
}

// verifyBlockHash checks if the data matches the expected CID hash.
func verifyBlockHash(c cid.Cid, data []byte) bool {
	ca := mcp.NewIPFSContentAddresser()
	computedCID := ca.Hash(data)
	return computedCID.Equals(c)
}

// detectContentType attempts to determine the content type from data.
func detectContentType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}

	// PDF
	if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
		return "application/pdf"
	}

	// JSON
	if data[0] == 0x7B || (data[0] == 0x09 || data[0] == 0x0A || data[0] == 0x0D || data[0] == 0x20) && data[1] == 0x7B {
		if isValidJSONStart(data) {
			return "application/json"
		}
	}

	// HTML
	if looksLikeHTML(data) {
		return "text/html; charset=utf-8"
	}

	// Plain text (if mostly ASCII)
	if isLikelyText(data) {
		return "text/plain; charset=utf-8"
	}

	return "application/octet-stream"
}

func isValidJSONStart(data []byte) bool {
	for _, b := range data {
		if b == '{' || b == '[' {
			return true
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return false
}

func looksLikeHTML(data []byte) bool {
	s := string(data)
	lower := strings.ToLower(s)
	return strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html")
}

func isLikelyText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	nonPrintable := 0
	for _, b := range data {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(data)) < 0.1
}

// handleHealth returns liveness status.
func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// handleReady returns readiness status.
func (g *Gateway) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// handleStats returns gateway statistics.
func (g *Gateway) handleStats(w http.ResponseWriter, r *http.Request) {
	// Stats are now provided by Prometheus metrics
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"use /metrics for Prometheus metrics"}`))
}

// handlePrometheusMetrics exposes Prometheus metrics endpoint.
func (g *Gateway) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// Prometheus handler is typically registered at the server level
	// This is a placeholder for the actual Prometheus handler
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# Prometheus metrics endpoint\n"))
}

// corsMiddleware handles CORS headers.
func (g *Gateway) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		allowed := false
		for _, ao := range g.config.AllowedOrigins {
			if ao == "*" || ao == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, X-CID")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware records request metrics.
func (g *Gateway) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		if g.metrics != nil {
			g.metrics.ActiveRequests.Inc()
		}
		next.ServeHTTP(wrapped, r)
		if g.metrics != nil {
			g.metrics.ActiveRequests.Dec()
		}

		duration := time.Since(start)
		path := r.URL.Path
		if len(path) > 50 {
			path = path[:50] + "..."
		}

		// Record metrics
		if g.metrics != nil {
			status := strconv.Itoa(wrapped.statusCode)
			g.metrics.RequestsTotal.WithLabelValues(r.Method, status, path).Inc()
			g.metrics.RequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// IPFSURIResolver resolves ipfs:// URIs to HTTP URLs.
type IPFSURIResolver struct {
	gatewayURL string
}

// NewIPFSURIResolver creates a new URI resolver.
func NewIPFSURIResolver(gatewayURL string) *IPFSURIResolver {
	return &IPFSURIResolver{gatewayURL: gatewayURL}
}

// Resolve converts an ipfs:// URI to an HTTP URL.
func (r *IPFSURIResolver) Resolve(uri string) (string, error) {
	if strings.HasPrefix(uri, "ipfs://") {
		cidStr := strings.TrimPrefix(uri, "ipfs://")
		parts := strings.SplitN(cidStr, "/", 2)
		cidPart := parts[0]
		pathPart := ""
		if len(parts) > 1 {
			pathPart = "/" + parts[1]
		}

		if _, err := cid.Decode(cidStr); err != nil {
			return "", fmt.Errorf("invalid CID in URI: %w", err)
		}

		return fmt.Sprintf("%s/ipfs/%s%s", r.gatewayURL, cidPart, pathPart), nil
	}

	return "", fmt.Errorf("unsupported URI scheme: %s", uri)
}
