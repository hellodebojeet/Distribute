package observability

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics provides metrics collection
type Metrics interface {
	// Counter increments a counter metric
	Counter(name string, labels map[string]string) Counter

	// Gauge sets a gauge metric
	Gauge(name string, labels map[string]string) Gauge

	// Histogram observes a histogram metric
	Histogram(name string, labels map[string]string) Histogram

	// StartServer starts the metrics HTTP server
	StartServer(addr string) error

	// StopServer stops the metrics HTTP server
	StopServer() error
}

// Counter represents a counter metric
type Counter interface {
	// Inc increments the counter by 1
	Inc()

	// Add adds the given value to the counter
	Add(float64)
}

// Gauge represents a gauge metric
type Gauge interface {
	// Set sets the gauge to the given value
	Set(float64)

	// Inc increments the gauge by 1
	Inc()

	// Dec decrements the gauge by 1
	Dec()

	// Add adds the given value to the gauge
	Add(float64)

	// Sub subtracts the given value from the gauge
	Sub(float64)
}

// Histogram represents a histogram metric
type Histogram interface {
	// Observe adds a single observation to the histogram
	Observe(float64)
}

// prometheusMetrics implements Metrics
type prometheusMetrics struct {
	registry   *prometheus.Registry
	server     *http.Server
	mu         sync.Mutex
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// MetricsConfig holds configuration for metrics
type MetricsConfig struct {
	Namespace string
	Subsystem string
}

// NewMetrics creates a new metrics instance
func NewMetrics(cfg MetricsConfig) Metrics {
	return &prometheusMetrics{
		registry:   prometheus.NewRegistry(),
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func (m *prometheusMetrics) Counter(name string, labels map[string]string) Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if vec, exists := m.counters[key]; exists {
		return vec.With(labels)
	}

	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: name + " counter",
	}, labelKeys(labels))

	m.registry.MustRegister(vec)
	m.counters[key] = vec

	return vec.With(labels)
}

func (m *prometheusMetrics) Gauge(name string, labels map[string]string) Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if vec, exists := m.gauges[key]; exists {
		return vec.With(labels)
	}

	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: name + " gauge",
	}, labelKeys(labels))

	m.registry.MustRegister(vec)
	m.gauges[key] = vec

	return vec.With(labels)
}

func (m *prometheusMetrics) Histogram(name string, labels map[string]string) Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name
	if vec, exists := m.histograms[key]; exists {
		return vec.With(labels)
	}

	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    name + " histogram",
		Buckets: prometheus.DefBuckets,
	}, labelKeys(labels))

	m.registry.MustRegister(vec)
	m.histograms[key] = vec

	return vec.With(labels)
}

func (m *prometheusMetrics) StartServer(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	m.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error
		}
	}()

	return nil
}

func (m *prometheusMetrics) StopServer() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

func labelKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	return keys
}

// NoopMetrics is a metrics implementation that does nothing
type NoopMetrics struct{}

func (m *NoopMetrics) Counter(name string, labels map[string]string) Counter {
	return &noopCounter{}
}

func (m *NoopMetrics) Gauge(name string, labels map[string]string) Gauge {
	return &noopGauge{}
}

func (m *NoopMetrics) Histogram(name string, labels map[string]string) Histogram {
	return &noopHistogram{}
}

func (m *NoopMetrics) StartServer(addr string) error { return nil }
func (m *NoopMetrics) StopServer() error             { return nil }

type noopCounter struct{}

func (c *noopCounter) Inc()        {}
func (c *noopCounter) Add(float64) {}

type noopGauge struct{}

func (g *noopGauge) Set(float64) {}
func (g *noopGauge) Inc()        {}
func (g *noopGauge) Dec()        {}
func (g *noopGauge) Add(float64) {}
func (g *noopGauge) Sub(float64) {}

type noopHistogram struct{}

func (h *noopHistogram) Observe(float64) {}
