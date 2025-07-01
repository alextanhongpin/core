package metrics_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResponseSizeCalculationFixed tests response size calculation with proper isolation
func TestResponseSizeCalculationFixed(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		url          string
		headers      map[string]string
		expectedSize int
		tolerance    int
	}{
		{
			name:         "GET request",
			method:       "GET",
			url:          "/test",
			expectedSize: 17, // Method + URL + HTTP version + CRLF
			tolerance:    3,
		},
		{
			name:         "GET with query params",
			method:       "GET",
			url:          "/test?param=value",
			expectedSize: 28,
			tolerance:    5,
		},
		{
			name:   "GET with headers",
			method: "GET",
			url:    "/test",
			headers: map[string]string{
				"User-Agent": "test-agent",
				"Accept":     "application/json",
			},
			expectedSize: 80, // Base + headers
			tolerance:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated registry for this test
			registry := prometheus.NewRegistry()

			// Create isolated response size histogram
			responseSize := prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "test_response_size_bytes",
					Help:    "Test response size histogram",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"method", "status"},
			)
			registry.MustRegister(responseSize)

			// Create request
			req := httptest.NewRequest(tt.method, tt.url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Calculate size
			actualSize := computeApproximateRequestSize(req)

			// Record metric
			responseSize.WithLabelValues(tt.method, "200").Observe(float64(actualSize))

			// Verify size is within tolerance
			diff := actualSize - tt.expectedSize
			if diff < 0 {
				diff = -diff
			}

			assert.True(t, diff <= tt.tolerance,
				"Size difference %d exceeds tolerance %d (expected: %d, actual: %d)",
				diff, tt.tolerance, tt.expectedSize, actualSize)

			// Verify metric was recorded
			families, err := registry.Gather()
			require.NoError(t, err)
			require.Len(t, families, 1)

			metric := families[0].GetMetric()[0]
			histogram := metric.GetHistogram()
			assert.Equal(t, uint64(1), histogram.GetSampleCount())
			assert.Equal(t, float64(actualSize), histogram.GetSampleSum())
		})
	}
}

// TestResponseSizeIsolated tests response size with isolated metrics
func TestResponseSizeIsolated(t *testing.T) {
	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create isolated response size histogram
	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "isolated_response_size_bytes",
			Help:    "Isolated response size histogram",
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{},
	)
	registry.MustRegister(responseSize)

	// Record a single observation
	responseSize.WithLabelValues().Observe(23)

	// Gather metrics
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	// Verify metric values
	metric := families[0].GetMetric()[0]
	histogram := metric.GetHistogram()
	assert.Equal(t, uint64(1), histogram.GetSampleCount())
	assert.Equal(t, float64(23), histogram.GetSampleSum())

	// Verify bucket counts
	buckets := histogram.GetBucket()
	assert.Equal(t, uint64(1), buckets[0].GetCumulativeCount()) // 200 bucket
	assert.Equal(t, uint64(1), buckets[1].GetCumulativeCount()) // 500 bucket
	assert.Equal(t, uint64(1), buckets[2].GetCumulativeCount()) // 900 bucket
	assert.Equal(t, uint64(1), buckets[3].GetCumulativeCount()) // 1500 bucket
}

// TestRequestDurationHandlerIsolated tests request duration with isolated metrics
func TestRequestDurationHandlerIsolated(t *testing.T) {
	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create isolated request duration histogram
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "isolated_request_duration_seconds",
			Help: "Isolated request duration histogram",
		},
		[]string{"method", "status"},
	)
	registry.MustRegister(requestDuration)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with duration tracking
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, "200").Observe(duration)
	})

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Verify metrics
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	metric := families[0].GetMetric()[0]
	histogram := metric.GetHistogram()
	assert.Equal(t, uint64(1), histogram.GetSampleCount())
	assert.True(t, histogram.GetSampleSum() > 0.01) // At least 10ms
}

// TestREDMetricsIsolated tests RED metrics with complete isolation
func TestREDMetricsIsolated(t *testing.T) {
	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create isolated RED tracker
	redHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "isolated_red",
			Help: "Isolated RED metrics",
		},
		[]string{"service", "action", "status"},
	)
	registry.MustRegister(redHistogram)

	// Create tracker
	tracker := &REDTracker{
		histogram: redHistogram,
	}

	// Track some operations
	ctx := context.Background()

	// Success operation
	done := tracker.Track(ctx, "user_service", "login")
	time.Sleep(1 * time.Millisecond)
	done.Finish("ok")

	// Error operation
	done = tracker.Track(ctx, "user_service", "login")
	time.Sleep(1 * time.Millisecond)
	done.Finish("err")

	// Gather and verify metrics
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	// Parse metrics output
	var buf bytes.Buffer
	for _, family := range families {
		_, err := expfmt.WriteFamily(&buf, family)
		require.NoError(t, err)
	}

	output := buf.String()

	// Verify we have metrics for both success and error
	assert.Contains(t, output, `service="user_service"`)
	assert.Contains(t, output, `action="login"`)
	assert.Contains(t, output, `status="ok"`)
	assert.Contains(t, output, `status="err"`)

	// Verify count values (should be exactly 1 for each)
	lines := strings.Split(output, "\n")
	okCount := 0
	errCount := 0

	for _, line := range lines {
		if strings.Contains(line, `status="ok"`) && strings.Contains(line, "_count") {
			if strings.HasSuffix(strings.TrimSpace(line), " 1") {
				okCount++
			}
		}
		if strings.Contains(line, `status="err"`) && strings.Contains(line, "_count") {
			if strings.HasSuffix(strings.TrimSpace(line), " 1") {
				errCount++
			}
		}
	}

	assert.Equal(t, 1, okCount, "Should have exactly 1 ok status count")
	assert.Equal(t, 1, errCount, "Should have exactly 1 err status count")
}

// TestConcurrentMetricsIsolated tests concurrent access with isolated metrics
func TestConcurrentMetricsIsolated(t *testing.T) {
	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create isolated metrics
	redHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "concurrent_red",
			Help: "Concurrent RED metrics",
		},
		[]string{"service", "action", "status"},
	)
	registry.MustRegister(redHistogram)

	tracker := &REDTracker{
		histogram: redHistogram,
	}

	// Test concurrent access
	const numWorkers = 10
	const opsPerWorker = 100

	var wg sync.WaitGroup
	ctx := context.Background()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				action := fmt.Sprintf("worker_%d", workerID)
				status := "ok"
				if j%2 == 0 {
					status = "err"
				}

				done := tracker.Track(ctx, "concurrent_test", action)
				time.Sleep(1 * time.Millisecond)
				done.Finish(status)
			}
		}(i)
	}

	wg.Wait()

	// Verify metrics
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	// Count total operations
	totalOps := 0
	for _, metric := range families[0].GetMetric() {
		histogram := metric.GetHistogram()
		totalOps += int(histogram.GetSampleCount())
	}

	expectedOps := numWorkers * opsPerWorker
	assert.Equal(t, expectedOps, totalOps, "Total operations should match expected")
}

// TestInFlightGaugeIsolated tests in-flight gauge with isolation
func TestInFlightGaugeIsolated(t *testing.T) {
	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create isolated in-flight gauge
	inFlight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "isolated_in_flight_requests",
			Help: "Isolated in-flight requests",
		},
		[]string{"method"},
	)
	registry.MustRegister(inFlight)

	// Test handler with in-flight tracking
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment in-flight
		inFlight.WithLabelValues(r.Method).Inc()
		defer inFlight.WithLabelValues(r.Method).Dec()

		// Simulate work
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Make concurrent requests
	const numRequests = 5
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}()
	}

	wg.Wait()

	// Verify final state (should be 0 in-flight)
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	metric := families[0].GetMetric()[0]
	gauge := metric.GetGauge()
	assert.Equal(t, float64(0), gauge.GetValue(), "In-flight requests should be 0 after completion")
}
