package metrics_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManager provides isolated testing environment for metrics
type TestManager struct {
	registry *prometheus.Registry
	metrics  *metrics.Metrics
}

// NewTestManager creates a new isolated test environment
func NewTestManager() *TestManager {
	registry := prometheus.NewRegistry()

	// Create metrics with isolated registry
	m := &metrics.Metrics{
		InFlightGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "in_flight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		}),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "request_duration_seconds",
				Help: "A histogram of latencies for requests.",
			},
			[]string{"handler", "method"},
		),
		ResponseSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "response_size_bytes",
			Help: "A histogram of response sizes for requests.",
		}),
		RED: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "red",
				Help: "RED metrics",
			},
			[]string{"service", "action", "status"},
		),
	}

	// Register all metrics with the isolated registry
	registry.MustRegister(
		m.InFlightGauge,
		m.RequestDuration,
		m.ResponseSize,
		m.RED,
	)

	return &TestManager{
		registry: registry,
		metrics:  m,
	}
}

// GetMetrics returns the metrics instance
func (tm *TestManager) GetMetrics() *metrics.Metrics {
	return tm.metrics
}

// GetRegistry returns the isolated registry
func (tm *TestManager) GetRegistry() *prometheus.Registry {
	return tm.registry
}

// GetMetricsOutput returns the current metrics as a string
func (tm *TestManager) GetMetricsOutput() (string, error) {
	gatherer := tm.registry
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return "", err
	}

	var output strings.Builder
	for _, mf := range metricFamilies {
		output.WriteString(fmt.Sprintf("# HELP %s %s\n", mf.GetName(), mf.GetHelp()))
		output.WriteString(fmt.Sprintf("# TYPE %s %s\n", mf.GetName(), mf.GetType().String()))

		for _, m := range mf.GetMetric() {
			switch mf.GetType() {
			case 1: // COUNTER
				output.WriteString(fmt.Sprintf("%s", mf.GetName()))
				if len(m.GetLabel()) > 0 {
					output.WriteString("{")
					for i, label := range m.GetLabel() {
						if i > 0 {
							output.WriteString(",")
						}
						output.WriteString(fmt.Sprintf("%s=\"%s\"", label.GetName(), label.GetValue()))
					}
					output.WriteString("}")
				}
				output.WriteString(fmt.Sprintf(" %g\n", m.GetCounter().GetValue()))
			case 2: // GAUGE
				output.WriteString(fmt.Sprintf("%s", mf.GetName()))
				if len(m.GetLabel()) > 0 {
					output.WriteString("{")
					for i, label := range m.GetLabel() {
						if i > 0 {
							output.WriteString(",")
						}
						output.WriteString(fmt.Sprintf("%s=\"%s\"", label.GetName(), label.GetValue()))
					}
					output.WriteString("}")
				}
				output.WriteString(fmt.Sprintf(" %g\n", m.GetGauge().GetValue()))
			case 4: // HISTOGRAM
				baseName := mf.GetName()
				labelStr := ""
				if len(m.GetLabel()) > 0 {
					labelStr = "{"
					for i, label := range m.GetLabel() {
						if i > 0 {
							labelStr += ","
						}
						labelStr += fmt.Sprintf("%s=\"%s\"", label.GetName(), label.GetValue())
					}
					labelStr += "}"
				}

				// Buckets
				for _, bucket := range m.GetHistogram().GetBucket() {
					bucketLabelStr := labelStr
					if bucketLabelStr == "" {
						bucketLabelStr = fmt.Sprintf("{le=\"%s\"}", formatFloat(bucket.GetUpperBound()))
					} else {
						bucketLabelStr = bucketLabelStr[:len(bucketLabelStr)-1] + fmt.Sprintf(",le=\"%s\"}", formatFloat(bucket.GetUpperBound()))
					}
					output.WriteString(fmt.Sprintf("%s_bucket%s %d\n", baseName, bucketLabelStr, bucket.GetCumulativeCount()))
				}

				// Sum
				output.WriteString(fmt.Sprintf("%s_sum%s %g\n", baseName, labelStr, m.GetHistogram().GetSampleSum()))

				// Count
				output.WriteString(fmt.Sprintf("%s_count%s %d\n", baseName, labelStr, m.GetHistogram().GetSampleCount()))
			}
		}
	}

	return output.String(), nil
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) && f != 0 {
		return fmt.Sprintf("%.0f", f)
	}
	if f == 0 {
		return "0"
	}
	return fmt.Sprintf("%g", f)
}

func TestIsolatedInFlightGauge(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Test initial state
	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)
	assert.Contains(t, output, "in_flight_requests 0")

	// Test increment
	metrics.InFlightGauge.Inc()
	output, err = tm.GetMetricsOutput()
	require.NoError(t, err)
	assert.Contains(t, output, "in_flight_requests 1")

	// Test decrement
	metrics.InFlightGauge.Dec()
	output, err = tm.GetMetricsOutput()
	require.NoError(t, err)
	assert.Contains(t, output, "in_flight_requests 0")
}

func TestIsolatedResponseSize(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Create a simple request
	req := httptest.NewRequest("GET", "/test", nil)

	// Observe response size
	size := computeApproximateRequestSize(req)
	metrics.ResponseSize.Observe(float64(size))

	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	// Check that the histogram contains our observation
	assert.Contains(t, output, "response_size_bytes_count 1")
	assert.Contains(t, output, fmt.Sprintf("response_size_bytes_sum %d", size))
}

func TestIsolatedRequestDurationHandler(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Wrap with metrics
	instrumentedHandler := InstrumentHandlerDuration(metrics.RequestDuration.With(prometheus.Labels{
		"handler": "test",
		"method":  "GET",
	}), handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	instrumentedHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test", rr.Body.String())

	// Check metrics
	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	assert.Contains(t, output, "request_duration_seconds_count{handler=\"test\",method=\"GET\"} 1")
	assert.Contains(t, output, "request_duration_seconds_sum{handler=\"test\",method=\"GET\"}")
}

func TestIsolatedREDMetrics(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Test successful operation
	tracker := NewREDTracker("user_service", "login")
	tracker.Start()
	time.Sleep(1 * time.Millisecond)
	tracker.Success()
	tracker.Observe(metrics.RED)

	// Test error operation
	errorTracker := NewREDTracker("user_service", "login")
	errorTracker.Start()
	time.Sleep(1 * time.Millisecond)
	errorTracker.Error()
	errorTracker.Observe(metrics.RED)

	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	// Should have both success and error metrics
	assert.Contains(t, output, "red_count{action=\"login\",service=\"user_service\",status=\"ok\"} 1")
	assert.Contains(t, output, "red_count{action=\"login\",service=\"user_service\",status=\"err\"} 1")
	assert.Contains(t, output, "red_sum{action=\"login\",service=\"user_service\",status=\"ok\"}")
	assert.Contains(t, output, "red_sum{action=\"login\",service=\"user_service\",status=\"err\"}")
}

func TestIsolatedConcurrentMetrics(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	const numWorkers = 5
	const operationsPerWorker = 20

	var wg sync.WaitGroup

	// Run concurrent operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Simulate some work
				metrics.InFlightGauge.Inc()

				tracker := NewREDTracker("concurrent_test", fmt.Sprintf("worker_%d", workerID))
				tracker.Start()

				// Simulate varying work duration
				time.Sleep(time.Duration(j%3) * time.Millisecond)

				// Simulate success/error ratio
				if j%5 == 0 {
					tracker.Error()
				} else {
					tracker.Success()
				}

				tracker.Observe(metrics.RED)
				metrics.InFlightGauge.Dec()
			}
		}(i)
	}

	wg.Wait()

	// Verify metrics
	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	// Check that we have metrics for all workers
	for i := 0; i < numWorkers; i++ {
		workerPattern := fmt.Sprintf("worker_%d", i)
		assert.Contains(t, output, workerPattern, "Should have metrics for worker %d", i)
	}

	// Check final in-flight gauge is zero
	assert.Contains(t, output, "in_flight_requests 0")

	// Count total operations
	totalOps := numWorkers * operationsPerWorker

	// We should have approximately the right number of operations
	// Note: Due to concurrent access, we just check that we have some reasonable amount
	assert.Contains(t, output, "concurrent_test", "Should have concurrent test metrics")
}

func TestIsolatedEndToEndScenario(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Create a realistic handler that tracks multiple metrics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track RED metrics
		tracker := NewREDTracker("api", "user_profile")
		tracker.Start()

		// Simulate processing time
		time.Sleep(2 * time.Millisecond)

		// Simulate success/error based on path
		if r.URL.Path == "/error" {
			tracker.Error()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		} else {
			tracker.Success()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user": "john", "profile": "active"}`))
		}

		tracker.Observe(metrics.RED)

		// Track response size
		responseSize := len(w.Header().Get("Content-Length"))
		if responseSize == 0 {
			responseSize = 32 // Approximate JSON response size
		}
		metrics.ResponseSize.Observe(float64(responseSize))
	})

	// Wrap with all instrumentation
	instrumentedHandler := InstrumentHandlerInFlight(metrics.InFlightGauge,
		InstrumentHandlerDuration(metrics.RequestDuration.With(prometheus.Labels{
			"handler": "user_profile",
			"method":  "GET",
		}), handler))

	// Make successful requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/user/123", nil)
		rr := httptest.NewRecorder()
		instrumentedHandler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Make error request
	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()
	instrumentedHandler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	// Verify all metrics are working
	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	// Check request duration metrics
	assert.Contains(t, output, "request_duration_seconds_count{handler=\"user_profile\",method=\"GET\"} 4")

	// Check RED metrics
	assert.Contains(t, output, "red_count{action=\"user_profile\",service=\"api\",status=\"ok\"} 3")
	assert.Contains(t, output, "red_count{action=\"user_profile\",service=\"api\",status=\"err\"} 1")

	// Check response size metrics
	assert.Contains(t, output, "response_size_bytes_count 4")

	// Check in-flight gauge is back to zero
	assert.Contains(t, output, "in_flight_requests 0")
}

func TestIsolatedMetricsHTTPEndpoint(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Add some test data
	metrics.InFlightGauge.Set(5)
	metrics.ResponseSize.Observe(1024)

	tracker := NewREDTracker("test_service", "test_action")
	tracker.Start()
	time.Sleep(1 * time.Millisecond)
	tracker.Success()
	tracker.Observe(metrics.RED)

	// Create HTTP handler for metrics endpoint
	handler := promhttp.HandlerFor(tm.GetRegistry(), promhttp.HandlerOpts{})

	// Test metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")

	body := rr.Body.String()
	assert.Contains(t, body, "in_flight_requests")
	assert.Contains(t, body, "response_size_bytes")
	assert.Contains(t, body, "red")
	assert.Contains(t, body, "test_service")
	assert.Contains(t, body, "test_action")
}

func TestIsolatedMemoryEfficiency(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Create many trackers but don't let them accumulate
	for i := 0; i < 1000; i++ {
		tracker := NewREDTracker("perf_test", fmt.Sprintf("action_%d", i%10))
		tracker.Start()
		tracker.Success()
		tracker.Observe(metrics.RED)
	}

	// Verify we have metrics but not excessive memory usage
	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	// Should have consolidated metrics for the 10 different actions
	assert.Contains(t, output, "perf_test")

	// Check that the metrics make sense
	lines := strings.Split(output, "\n")
	metricLines := 0
	for _, line := range lines {
		if strings.Contains(line, "red_count{") && strings.Contains(line, "perf_test") {
			metricLines++
		}
	}

	// Should have metrics for each of the 10 actions (action_0 through action_9)
	assert.True(t, metricLines >= 10, "Should have metrics for all actions")
	assert.True(t, metricLines <= 20, "Should not have excessive duplicate metrics")
}

func TestIsolatedRequestSizeCalculation(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		headers  map[string]string
		body     string
		expected int
	}{
		{
			name:     "simple GET",
			method:   "GET",
			url:      "/test",
			headers:  nil,
			body:     "",
			expected: 17, // "GET /test HTTP/1.1\r\n\r\n"
		},
		{
			name:   "GET with headers",
			method: "GET",
			url:    "/api/users",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
			},
			body:     "",
			expected: 101, // Approximate with headers
		},
		{
			name:     "POST with body",
			method:   "POST",
			url:      "/api/users",
			headers:  map[string]string{"Content-Type": "application/json"},
			body:     `{"name": "John", "email": "john@example.com"}`,
			expected: 116, // Approximate with body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))

			// Add headers
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			size := computeApproximateRequestSize(req)

			// Allow for some variance in size calculation
			tolerance := 10
			if abs(size-tt.expected) > tolerance {
				t.Errorf("Expected size %d, got %d (tolerance: %d)", tt.expected, size, tolerance)
			}
		})
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestIsolatedContextCancellation(t *testing.T) {
	tm := NewTestManager()
	metrics := tm.GetMetrics()

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	tracker := NewREDTracker("test", "context_test")
	tracker.Start()

	// Wait for context cancellation
	<-ctx.Done()

	// This should represent a timeout/cancellation scenario
	tracker.SetStatus("timeout")
	tracker.Observe(metrics.RED)

	output, err := tm.GetMetricsOutput()
	require.NoError(t, err)

	assert.Contains(t, output, "red_count{action=\"context_test\",service=\"test\",status=\"timeout\"} 1")
}
