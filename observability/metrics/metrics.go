// Package metrics provides Prometheus metrics for the distributed filesystem.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace for all metrics.
const namespace = "distribute"

// BlockstoreMetrics holds metrics for blockstore operations.
type BlockstoreMetrics struct {
	BlocksTotal    prometheus.Counter
	BlocksPinned   prometheus.Gauge
	BlockGetOps    prometheus.Counter
	BlockPutOps    prometheus.Counter
	BlockGetErrors prometheus.Counter
	BlockSize      prometheus.Histogram
}

// NewBlockstoreMetrics creates and registers blockstore metrics.
func NewBlockstoreMetrics() *BlockstoreMetrics {
	return &BlockstoreMetrics{
		BlocksTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "blocks_total",
			Help:      "Total number of blocks stored",
		}),
		BlocksPinned: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "pinned_total",
			Help:      "Number of pinned blocks",
		}),
		BlockGetOps: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "get_total",
			Help:      "Total number of block get operations",
		}),
		BlockPutOps: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "put_total",
			Help:      "Total number of block put operations",
		}),
		BlockGetErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "get_errors_total",
			Help:      "Total number of block get errors",
		}),
		BlockSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "blockstore",
			Name:      "size_bytes",
			Help:      "Size of blocks in bytes",
			Buckets:   []float64{256, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304},
		}),
	}
}

// DHTMetrics holds metrics for DHT operations.
type DHTMetrics struct {
	LookupsTotal   prometheus.Counter
	ProvideOps     prometheus.Counter
	FindProviders  prometheus.Counter
	RoutingTable   prometheus.Gauge
	LookupDuration prometheus.Histogram
}

// NewDHTMetrics creates and registers DHT metrics.
func NewDHTMetrics() *DHTMetrics {
	return &DHTMetrics{
		LookupsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "dht",
			Name:      "lookups_total",
			Help:      "Total number of DHT lookups",
		}),
		ProvideOps: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "dht",
			Name:      "provide_total",
			Help:      "Total number of provide operations",
		}),
		FindProviders: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "dht",
			Name:      "find_providers_total",
			Help:      "Total number of find providers operations",
		}),
		RoutingTable: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "dht",
			Name:      "routing_table_size",
			Help:      "Number of peers in DHT routing table",
		}),
		LookupDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "dht",
			Name:      "lookup_duration_seconds",
			Help:      "Duration of DHT lookups",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		}),
	}
}

// GatewayMetrics holds metrics for the HTTP gateway.
type GatewayMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
	BytesServed     prometheus.Counter
	CacheHits       prometheus.Counter
	CacheMisses     prometheus.Counter
}

// NewGatewayMetrics creates and registers gateway metrics.
func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "requests_total",
			Help:      "Total number of HTTP gateway requests",
		}, []string{"method", "status", "path"}),
		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP gateway requests",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
		}, []string{"method", "path"}),
		ActiveRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "active_requests",
			Help:      "Number of active HTTP requests",
		}),
		BytesServed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "bytes_served_total",
			Help:      "Total bytes served by the gateway",
		}),
		CacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "cache_hits_total",
			Help:      "Total cache hits",
		}),
		CacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "cache_misses_total",
			Help:      "Total cache misses",
		}),
	}
}

// BitswapMetrics holds metrics for block exchange (bitswap) operations.
type BitswapMetrics struct {
	BlocksSent     prometheus.Counter
	BlocksReceived prometheus.Counter
	WantlistSize   prometheus.Gauge
	MessagesSent   prometheus.Counter
	MessagesRecv   prometheus.Counter
}

// NewBitswapMetrics creates and registers bitswap metrics.
func NewBitswapMetrics() *BitswapMetrics {
	return &BitswapMetrics{
		BlocksSent: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "bitswap",
			Name:      "blocks_sent_total",
			Help:      "Total blocks sent to peers",
		}),
		BlocksReceived: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "bitswap",
			Name:      "blocks_received_total",
			Help:      "Total blocks received from peers",
		}),
		WantlistSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "bitswap",
			Name:      "wantlist_size",
			Help:      "Current wantlist size",
		}),
		MessagesSent: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "bitswap",
			Name:      "messages_sent_total",
			Help:      "Total bitswap messages sent",
		}),
		MessagesRecv: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "bitswap",
			Name:      "messages_received_total",
			Help:      "Total bitswap messages received",
		}),
	}
}

// ReplicationMetrics holds metrics for replication operations.
type ReplicationMetrics struct {
	ReplicationsTotal prometheus.Counter
	ReplicationErrors prometheus.Counter
	ReplicationFactor prometheus.Gauge
	UnderReplicated   prometheus.Gauge
	RepairOps         prometheus.Counter
}

// NewReplicationMetrics creates and registers replication metrics.
func NewReplicationMetrics() *ReplicationMetrics {
	return &ReplicationMetrics{
		ReplicationsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "replication",
			Name:      "total",
			Help:      "Total number of replication operations",
		}),
		ReplicationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "replication",
			Name:      "errors_total",
			Help:      "Total number of replication errors",
		}),
		ReplicationFactor: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "replication",
			Name:      "factor",
			Help:      "Current replication factor",
		}),
		UnderReplicated: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "replication",
			Name:      "under_replicated",
			Help:      "Number of under-replicated blocks",
		}),
		RepairOps: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "replication",
			Name:      "repair_total",
			Help:      "Total number of repair operations",
		}),
	}
}

// Collector aggregates all metrics for the system.
type Collector struct {
	Blockstore  *BlockstoreMetrics
	DHT         *DHTMetrics
	Gateway     *GatewayMetrics
	Bitswap     *BitswapMetrics
	Replication *ReplicationMetrics
}

// NewCollector creates a new metrics collector with all metric types.
func NewCollector() *Collector {
	return &Collector{
		Blockstore:  NewBlockstoreMetrics(),
		DHT:         NewDHTMetrics(),
		Gateway:     NewGatewayMetrics(),
		Bitswap:     NewBitswapMetrics(),
		Replication: NewReplicationMetrics(),
	}
}
