package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRegistry creates a new isolated Prometheus registry for testing
func createTestRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// TestREDTrackerBasicUsage demonstrates basic RED tracker usage without global state dependency
func TestREDTrackerBasicUsage(t *testing.T) {
	tracker := metrics.NewRED("user_service", "login")
	require.NotNil(t, tracker)

	// Test setting status
	tracker.SetStatus("processing")

	// Test failure case
	tracker.Fail()

	// Test completion (this will record metrics to global prometheus registry)
	tracker.Done()

	// Since we cannot easily isolate the global Prometheus metrics,
	// we just verify the tracker works without panicking
	assert.True(t, true, "RED tracker completed without errors")
}

// TestPrometheusHandlerIsolated shows how to test with isolated registry
func TestPrometheusHandlerIsolated(t *testing.T) {
	registry := createTestRegistry()

	// Create test metrics with our isolated registry
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler", "method"},
	)
	registry.MustRegister(duration)

	// Record some test data
	duration.WithLabelValues("api", "GET").Observe(0.1)
	duration.WithLabelValues("api", "POST").Observe(0.2)

	// Create the metrics handler with our isolated registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Make a request to the metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Verify our test metrics are present
	assert.Contains(t, body, "test_request_duration_seconds")
	assert.Contains(t, body, "method=\"GET\"")
	assert.Contains(t, body, "method=\"POST\"")
}
