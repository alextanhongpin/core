// Package health provides HTTP health check endpoints for monitoring and load balancers.
package health

import (
	"encoding/json"
	"maps"
	"net/http"
	"sync"
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

// Handler provides HTTP health check endpoints. It is safe for concurrent
// use; checks may be added while requests are served.
type Handler struct {
	version   string
	startTime time.Time

	mu       sync.RWMutex
	checkers map[string]Checker
}

// New creates a new health check handler.
func New(version string) *Handler {
	return &Handler{
		version:   version,
		startTime: time.Now(),
		checkers:  make(map[string]Checker),
	}
}

// AddCheck adds or replaces a named health check.
func (h *Handler) AddCheck(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// Live is a liveness probe: it always responds 200 while the process runs.
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

// Ready is a readiness probe. It runs all checks concurrently and responds:
//   - 200 "healthy" when every check is healthy (or there are none);
//   - 200 "degraded" when some check is degraded and none is unhealthy;
//   - 503 "unhealthy" when any check is unhealthy (or reports an unknown
//     status).
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	h.mu.RLock()
	checkers := maps.Clone(h.checkers)
	h.mu.RUnlock()

	response := Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Checks:    make(map[string]Check, len(checkers)),
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for name, checker := range checkers {
		wg.Go(func() {
			start := time.Now()
			check := checker()
			check.Latency = time.Since(start)

			mu.Lock()
			response.Checks[name] = check
			mu.Unlock()
		})
	}
	wg.Wait()

	for _, check := range response.Checks {
		switch check.Status {
		case StatusHealthy:
		case StatusDegraded:
			if response.Status == StatusHealthy {
				response.Status = StatusDegraded
			}
		default:
			response.Status = StatusUnhealthy
		}
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
