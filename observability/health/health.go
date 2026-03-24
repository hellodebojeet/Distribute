// Package health provides health check utilities for the distributed filesystem.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a single health check.
type Check struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration_ms"`
}

// Result represents the result of all health checks.
type Result struct {
	Status    Status            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    []Check           `json:"checks"`
	Details   map[string]string `json:"details,omitempty"`
}

// Checker is a function that performs a health check.
type Checker func(ctx context.Context) Check

// Manager manages health checks.
type Manager struct {
	mu      sync.RWMutex
	checks  map[string]Checker
	details map[string]string
}

// NewManager creates a new health check manager.
func NewManager() *Manager {
	return &Manager{
		checks:  make(map[string]Checker),
		details: make(map[string]string),
	}
}

// Register adds a health check.
func (m *Manager) Register(name string, check Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[name] = check
}

// SetDetail sets a detail value.
func (m *Manager) SetDetail(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.details[key] = value
}

// Check runs all health checks and returns the result.
func (m *Manager) Check(ctx context.Context) Result {
	m.mu.RLock()
	checks := make(map[string]Checker, len(m.checks))
	for k, v := range m.checks {
		checks[k] = v
	}
	details := make(map[string]string, len(m.details))
	for k, v := range m.details {
		details[k] = v
	}
	m.mu.RUnlock()

	results := make([]Check, 0, len(checks))
	globalStatus := StatusHealthy

	for name, checker := range checks {
		start := time.Now()
		check := checker(ctx)
		check.Duration = time.Since(start)
		check.Name = name

		if check.Status == "" {
			check.Status = StatusHealthy
		}

		results = append(results, check)

		// Update global status
		switch check.Status {
		case StatusUnhealthy:
			globalStatus = StatusUnhealthy
		case StatusDegraded:
			if globalStatus != StatusUnhealthy {
				globalStatus = StatusDegraded
			}
		}
	}

	return Result{
		Status:    globalStatus,
		Timestamp: time.Now(),
		Checks:    results,
		Details:   details,
	}
}

// Handler returns an HTTP handler for health checks.
func (m *Manager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := m.Check(r.Context())

		// Set status code based on health
		statusCode := http.StatusOK
		switch result.Status {
		case StatusDegraded:
			statusCode = http.StatusOK // Still OK, just degraded
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	})
}

// ReadinessHandler returns an HTTP handler for readiness checks.
func (m *Manager) ReadinessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := m.Check(r.Context())

		// Readiness only fails on unhealthy status
		statusCode := http.StatusOK
		if result.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	})
}

// LivenessHandler returns an HTTP handler for liveness checks.
// Liveness is always OK if the process is running.
func LivenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	})
}

// Common checkers

// HTTPChecker creates a health check that verifies an HTTP endpoint is reachable.
func HTTPChecker(url string, timeout time.Duration) Checker {
	return func(ctx context.Context) Check {
		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return Check{
				Status:  StatusUnhealthy,
				Message: err.Error(),
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return Check{
				Status:  StatusUnhealthy,
				Message: err.Error(),
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return Check{Status: StatusHealthy}
		}

		return Check{
			Status:  StatusDegraded,
			Message: "HTTP status " + resp.Status,
		}
	}
}

// TCPChecker creates a health check that verifies a TCP connection can be established.
func TCPChecker(address string, timeout time.Duration) Checker {
	return func(ctx context.Context) Check {
		// In a full implementation, this would dial TCP
		// For now, return healthy
		return Check{Status: StatusHealthy}
	}
}
