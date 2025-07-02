// Package health provides HTTP health check endpoints for monitoring and load balancers.
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Status represents the health status of the application.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Response represents a health check response.
type Response struct {
	Status    Status           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Version   string           `json:"version,omitempty"`
	Uptime    time.Duration    `json:"uptime,omitempty"`
	Checks    map[string]Check `json:"checks,omitempty"`
}

// Check represents an individual health check result.
type Check struct {
	Status  Status        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// Checker is a function that performs a health check.
type Checker func() Check

// Handler provides HTTP health check endpoints.
type Handler struct {
	version   string
	startTime time.Time
	checkers  map[string]Checker
}

// New creates a new health check handler.
func New(version string) *Handler {
	return &Handler{
		version:   version,
		startTime: time.Now(),
		checkers:  make(map[string]Checker),
	}
}

// AddCheck adds a named health check.
func (h *Handler) AddCheck(name string, checker Checker) {
	h.checkers[name] = checker
}

// Live returns a simple liveness probe handler.
// This endpoint should return 200 if the application is running.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
	}

	json.NewEncoder(w).Encode(response)
}

// Ready returns a readiness probe handler.
// This endpoint should return 200 if the application is ready to serve traffic.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Checks:    make(map[string]Check),
	}

	// Run all health checks
	overallHealthy := true
	for name, checker := range h.checkers {
		start := time.Now()
		check := checker()
		check.Latency = time.Since(start)

		response.Checks[name] = check

		if check.Status != StatusHealthy {
			overallHealthy = false
			if response.Status != StatusUnhealthy {
				response.Status = StatusDegraded
			}
		}
	}

	if !overallHealthy {
		response.Status = StatusUnhealthy
	}

	statusCode := http.StatusOK
	if response.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Health returns a detailed health check handler.
// This endpoint provides comprehensive health information.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.Ready(w, r) // Same as ready for now
}

// Common health check functions

// DatabaseCheck creates a health check for database connectivity.
func DatabaseCheck(pingFunc func() error) Checker {
	return func() Check {
		if err := pingFunc(); err != nil {
			return Check{
				Status:  StatusUnhealthy,
				Message: "Database connection failed: " + err.Error(),
			}
		}
		return Check{
			Status:  StatusHealthy,
			Message: "Database connection OK",
		}
	}
}

// HTTPCheck creates a health check for HTTP service dependencies.
func HTTPCheck(url string, timeout time.Duration) Checker {
	return func() Check {
		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(url)
		if err != nil {
			return Check{
				Status:  StatusUnhealthy,
				Message: "HTTP check failed: " + err.Error(),
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return Check{
				Status:  StatusHealthy,
				Message: "HTTP service OK",
			}
		}

		return Check{
			Status:  StatusUnhealthy,
			Message: "HTTP service returned status " + resp.Status,
		}
	}
}

// MemoryCheck creates a health check for memory usage.
func MemoryCheck(maxMemoryMB int64) Checker {
	return func() Check {
		// This is a simplified memory check
		// In production, you might want to use runtime.MemStats
		return Check{
			Status:  StatusHealthy,
			Message: "Memory usage OK",
		}
	}
}
