package metrics

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Global metrics with proper initialization
	once sync.Once

	InFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "in_flight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status", "version"},
	)

	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500, 5000, 15000, 50000},
		},
		[]string{"content_type"},
	)

	RED = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "red_duration_milliseconds",
			Help:    "RED metrics tracking Rate, Error, Duration",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"service", "action", "status"},
	)

	// Error counters for better observability
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "service", "action"},
	)
)

const (
	StatusOK       = "ok"
	StatusError    = "err"
	StatusTimeout  = "timeout"
	StatusPanic    = "panic"
	StatusCanceled = "canceled"
)

// REDConfig allows customization of RED metrics
type REDConfig struct {
	Service             string
	Action              string
	DefaultStatus       string
	EnablePanicRecovery bool
}

// REDTracker provides safe tracking of RED metrics with proper error handling
type REDTracker struct {
	config REDConfig
	status string
	start  time.Time
	mu     sync.RWMutex
}

// NewRED creates a new RED tracker with default configuration
func NewRED(service, action string) *REDTracker {
	return NewREDWithConfig(REDConfig{
		Service:             service,
		Action:              action,
		DefaultStatus:       StatusOK,
		EnablePanicRecovery: true,
	})
}

// NewREDWithConfig creates a new RED tracker with custom configuration
func NewREDWithConfig(config REDConfig) *REDTracker {
	if config.DefaultStatus == "" {
		config.DefaultStatus = StatusOK
	}

	tracker := &REDTracker{
		config: config,
		status: config.DefaultStatus,
		start:  time.Now(),
	}

	if config.EnablePanicRecovery {
		tracker.setupPanicRecovery()
	}

	return tracker
}

// setupPanicRecovery sets up automatic panic detection
func (r *REDTracker) setupPanicRecovery() {
	// This will be called if the function panics and Done() is deferred
	runtime.SetFinalizer(r, func(tracker *REDTracker) {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		if tracker.status == StatusOK {
			tracker.status = StatusPanic
		}
	})
}

// Done records the final metrics
func (r *REDTracker) Done() {
	r.mu.RLock()
	service := r.config.Service
	action := r.config.Action
	status := r.status
	duration := time.Since(r.start)
	r.mu.RUnlock()

	if service == "" || action == "" {
		ErrorsTotal.WithLabelValues("invalid_config", service, action).Inc()
		return
	}

	RED.WithLabelValues(service, action, status).
		Observe(float64(duration.Milliseconds()))

	if status != StatusOK {
		ErrorsTotal.WithLabelValues(status, service, action).Inc()
	}
}

// Fail marks the operation as failed
func (r *REDTracker) Fail() {
	r.SetStatus(StatusError)
}

// SetStatus sets a custom status
func (r *REDTracker) SetStatus(status string) {
	if status == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
}

// GetStatus returns the current status (thread-safe)
func (r *REDTracker) GetStatus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

// WithContext creates a tracker that respects context cancellation
func (r *REDTracker) WithContext(ctx context.Context) *REDTracker {
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			r.SetStatus(StatusTimeout)
		} else if ctx.Err() == context.Canceled {
			r.SetStatus(StatusCanceled)
		}
	}()
	return r
}

// RequestDurationHandler creates an HTTP middleware for tracking request duration
// with improved error handling and edge case coverage
func RequestDurationHandler(version string, next http.Handler) http.Handler {
	if version == "" {
		version = "unknown"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			http.Error(w, "nil request", http.StatusBadRequest)
			return
		}

		start := time.Now()
		wr := httputil.NewResponseWriterRecorder(w)

		// Handle panics
		defer func() {
			if rec := recover(); rec != nil {
				// Record panic metrics
				ErrorsTotal.WithLabelValues(StatusPanic, "http_handler", "request").Inc()

				duration := time.Since(start)
				method := safeString(r.Method, "UNKNOWN")
				path := extractPath(r)

				RequestDuration.
					WithLabelValues(method, path, "500", version).
					Observe(duration.Seconds())

				// Re-panic to let the server handle it
				panic(rec)
			}
		}()

		// Track in-flight requests
		InFlightGauge.Inc()
		defer InFlightGauge.Dec()

		next.ServeHTTP(wr, r)

		// Record metrics
		duration := time.Since(start)
		method := safeString(r.Method, "UNKNOWN")
		path := extractPath(r)
		status := fmt.Sprintf("%d", wr.StatusCode())

		RequestDuration.
			WithLabelValues(method, path, status, version).
			Observe(duration.Seconds())
	})
}

// ObserveResponseSize observes the response size with content type information
func ObserveResponseSize(r *http.Request, contentType string) int {
	if r == nil {
		ErrorsTotal.WithLabelValues("nil_request", "metrics", "response_size").Inc()
		return 0
	}

	size := computeApproximateRequestSize(r)
	contentType = normalizeContentType(contentType)

	ResponseSize.WithLabelValues(contentType).Observe(float64(size))
	return size
}

// Enhanced response size observation with automatic content type detection
func ObserveResponseSizeAuto(r *http.Request, w http.ResponseWriter) int {
	contentType := w.Header().Get("Content-Type")
	return ObserveResponseSize(r, contentType)
}

// Helper functions for safer string operations
func safeString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func extractPath(r *http.Request) string {
	if r == nil {
		return "unknown"
	}

	if r.Pattern != "" {
		return tail(strings.Fields(r.Pattern))
	}

	if r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}

	return "unknown"
}

func normalizeContentType(contentType string) string {
	if contentType == "" {
		return "unknown"
	}

	// Extract main type (e.g., "application/json; charset=utf-8" -> "application/json")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	return strings.TrimSpace(contentType)
}

// computeApproximateRequestSize calculates request size with safety checks
func computeApproximateRequestSize(r *http.Request) int {
	if r == nil {
		return 0
	}

	s := 0

	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)

	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}

	s += len(r.Host)

	if r.ContentLength > 0 {
		s += int(r.ContentLength)
	}

	return s
}

func tail(vs []string) string {
	if len(vs) == 0 {
		return ""
	}
	return vs[len(vs)-1]
}

// InitializeMetrics ensures metrics are registered only once
func InitializeMetrics(registry *prometheus.Registry) {
	once.Do(func() {
		if registry != nil {
			registry.MustRegister(
				InFlightGauge,
				RequestDuration,
				ResponseSize,
				RED,
				ErrorsTotal,
			)
		}
	})
}

// MetricsConfig allows customization of metric collection
type MetricsConfig struct {
	EnableInFlight     bool
	EnableDuration     bool
	EnableResponseSize bool
	EnableRED          bool
	EnableErrors       bool
	Version            string
	ServiceName        string
}

// DefaultMetricsConfig returns a configuration with all metrics enabled
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		EnableInFlight:     true,
		EnableDuration:     true,
		EnableResponseSize: true,
		EnableRED:          true,
		EnableErrors:       true,
		Version:            "1.0.0",
		ServiceName:        "unknown",
	}
}

// ConfigurableMiddleware creates a middleware with configurable metrics
func ConfigurableMiddleware(config MetricsConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var red *REDTracker
		if config.EnableRED {
			red = NewRED(config.ServiceName, "http_request")
			defer red.Done()
		}

		start := time.Now()

		if config.EnableInFlight {
			InFlightGauge.Inc()
			defer InFlightGauge.Dec()
		}

		wr := httputil.NewResponseWriterRecorder(w)

		next.ServeHTTP(wr, r)

		// Record metrics based on configuration
		if config.EnableDuration {
			duration := time.Since(start)
			method := safeString(r.Method, "UNKNOWN")
			path := extractPath(r)
			status := fmt.Sprintf("%d", wr.StatusCode())

			RequestDuration.
				WithLabelValues(method, path, status, config.Version).
				Observe(duration.Seconds())
		}

		if config.EnableResponseSize {
			ObserveResponseSizeAuto(r, wr)
		}

		if config.EnableRED && red != nil {
			if wr.StatusCode() >= 400 {
				red.Fail()
			}
		}
	})
}
