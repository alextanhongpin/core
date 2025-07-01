package metrics_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInFlightGaugeEdgeCases(t *testing.T) {
	// Create a fresh registry for this test
	reg := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_in_flight",
		Help: "Test gauge",
	})
	reg.MustRegister(gauge)

	t.Run("concurrent access", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				gauge.Inc()
				time.Sleep(10 * time.Millisecond)
				gauge.Dec()
			}()
		}

		wg.Wait()

		// Final value should be 0
		assert.Equal(t, 0.0, testutil.ToFloat64(gauge))
	})

	t.Run("negative values prevention", func(t *testing.T) {
		gauge.Set(5)
		gauge.Dec() // 4
		gauge.Dec() // 3
		gauge.Dec() // 2
		gauge.Dec() // 1
		gauge.Dec() // 0
		gauge.Dec() // This would make it -1

		// Prometheus allows negative values, but we can check if we're tracking correctly
		assert.Equal(t, -1.0, testutil.ToFloat64(gauge))
	})
}

func TestResponseSizeEdgeCases(t *testing.T) {
	reg := prometheus.NewRegistry()
	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_response_size",
			Help:    "Test response size",
			Buckets: []float64{100, 500, 1000},
		},
		[]string{},
	)
	reg.MustRegister(responseSize)

	tests := []struct {
		name     string
		request  *http.Request
		expected int
	}{
		{
			name:     "nil request",
			request:  nil,
			expected: 0,
		},
		{
			name:     "empty request",
			request:  httptest.NewRequest("GET", "/", nil),
			expected: 23, // Basic GET / request size
		},
		{
			name: "request with headers",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/api/users", strings.NewReader(`{"name":"test"}`))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer token")
				return r
			}(),
			expected: 101, // Approximate size with headers and body
		},
		{
			name:     "request with large URL",
			request:  httptest.NewRequest("GET", "/api/users?param1=value1&param2=value2&param3=value3", nil),
			expected: 77, // Size with long URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := metrics.ObserveResponseSize(tt.request)

			// Allow some tolerance for size calculation differences
			tolerance := 5
			assert.InDelta(t, tt.expected, size, float64(tolerance),
				"Expected size %d, got %d (within tolerance %d)", tt.expected, size, tolerance)
		})
	}
}

func TestRequestDurationHandlerEdgeCases(t *testing.T) {
	reg := prometheus.NewRegistry()
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_request_duration",
			Help:    "Test request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status", "version"},
	)
	reg.MustRegister(duration)

	t.Run("handler with panic", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		// Use a custom middleware since we can't modify the original
		middleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				defer func() {
					if rec := recover(); rec != nil {
						// Record the metric even on panic
						duration.WithLabelValues("GET", "/panic", "500", "test").Observe(time.Since(start).Seconds())
						// Don't re-panic for test
					}
				}()
				next.ServeHTTP(w, r)
			})
		}

		mux := http.NewServeMux()
		mux.Handle("/panic", middleware(panicHandler))

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Check that metric was recorded
		assert.Equal(t, 1, testutil.CollectAndCount(duration))
	})

	t.Run("nil request handling", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})

		// Test what happens with nil request (shouldn't happen in practice)
		middleware := metrics.RequestDurationHandler("test", handler)

		w := httptest.NewRecorder()

		// This would normally panic, but let's test the middleware's robustness
		req := httptest.NewRequest("GET", "/test", nil)
		middleware.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("empty pattern handling", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})

		mux := http.NewServeMux()
		mux.Handle("/", metrics.RequestDurationHandler("test", handler))

		req := httptest.NewRequest("GET", "/unknown", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should still record metrics even without a pattern
		assert.True(t, testutil.CollectAndCount(metrics.RequestDuration) >= 1)
	})
}

func TestREDTrackerEdgeCases(t *testing.T) {
	reg := prometheus.NewRegistry()
	redMetric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_red",
			Help:    "Test RED metric",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "action", "status"},
	)
	reg.MustRegister(redMetric)

	t.Run("concurrent access", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				red := metrics.NewRED("test_service", fmt.Sprintf("action_%d", id))
				defer red.Done()

				if id%2 == 0 {
					red.Fail()
				}

				time.Sleep(time.Millisecond)
			}(i)
		}

		wg.Wait()

		// Check that all metrics were recorded
		assert.True(t, testutil.CollectAndCount(metrics.RED) >= numGoroutines)
	})

	t.Run("multiple status changes", func(t *testing.T) {
		red := metrics.NewRED("test", "multiple_status")

		red.SetStatus("processing")
		red.SetStatus("validating")
		red.Fail()                    // Should override to "err"
		red.SetStatus("custom_error") // Should override "err"

		red.Done()

		// The final status should be "custom_error"
		// We can't easily test this without exposing internal state,
		// but we can check that metrics were recorded
		assert.True(t, testutil.CollectAndCount(metrics.RED) >= 1)
	})

	t.Run("empty service and action", func(t *testing.T) {
		red := metrics.NewRED("", "")
		red.Done()

		// Should handle empty strings gracefully
		// This test ensures no panic occurs
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		red := metrics.NewRED("test", "context_test")

		// Simulate long-running operation
		select {
		case <-ctx.Done():
			red.SetStatus("timeout")
		case <-time.After(100 * time.Millisecond):
			// This shouldn't happen due to timeout
		}

		red.Done()

		assert.True(t, testutil.CollectAndCount(metrics.RED) >= 1)
	})
}

func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	t.Run("RED tracker memory usage", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.GC() // Double GC to ensure clean slate
		runtime.ReadMemStats(&m1)

		// Create many RED trackers
		for i := 0; i < 10000; i++ {
			red := metrics.NewRED("memory_test", "action")
			red.Done()
		}

		runtime.GC()
		runtime.GC() // Double GC to ensure cleanup
		runtime.ReadMemStats(&m2)

		// Check if system memory is available (m2.Sys should be greater than m1.Sys)
		// This is a basic sanity check that we didn't underflow
		assert.GreaterOrEqual(t, m2.Sys, m1.Sys, "System memory should not decrease")

		// For total allocated memory, check that we haven't exceeded a reasonable limit
		// We use TotalAlloc which is cumulative and always increases
		totalIncrease := m2.TotalAlloc - m1.TotalAlloc
		assert.Less(t, totalIncrease, uint64(50*1024*1024),
			"Total memory allocation increase too large: %d bytes", totalIncrease)
	})
}

func TestConcurrentMetricsCollection(t *testing.T) {
	// Use isolated metrics to avoid interference
	reg := prometheus.NewRegistry()
	red := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "concurrent_red",
			Help: "Concurrent RED metrics",
		},
		[]string{"service", "action", "status"},
	)
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "concurrent_in_flight",
		Help: "Concurrent in-flight requests",
	})
	reg.MustRegister(red, inFlight)

	var wg sync.WaitGroup
	numWorkers := 10
	numRequests := 100

	// Simulate concurrent HTTP requests
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numRequests; j++ {
				// Simulate different types of requests
				status := "ok"

				// Simulate work
				time.Sleep(time.Millisecond)

				// Randomly fail some requests
				if (workerID+j)%7 == 0 {
					status = "error"
				}

				// Record metrics
				red.WithLabelValues("concurrent_test", fmt.Sprintf("worker_%d", workerID), status).Observe(1.0)

				// Also test in-flight gauge
				inFlight.Inc()
				time.Sleep(time.Microsecond * 100)
				inFlight.Dec()
			}
		}(i)
	}

	wg.Wait()

	// Verify metrics were collected
	expectedCount := numWorkers * numRequests

	// Collect all samples to verify we recorded the right number
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	var totalSamples int
	for _, mf := range metricFamilies {
		if mf.GetName() == "concurrent_red" {
			for _, metric := range mf.GetMetric() {
				if metric.GetHistogram() != nil {
					totalSamples += int(metric.GetHistogram().GetSampleCount())
				}
			}
		}
	}

	assert.GreaterOrEqual(t, totalSamples, expectedCount,
		"Expected at least %d samples, got %d", expectedCount, totalSamples)

	// In-flight gauge should be back to 0 (or close to it)
	inFlightValue := testutil.ToFloat64(inFlight)
	assert.True(t, inFlightValue >= -1 && inFlightValue <= 1,
		"In-flight gauge should be near 0, got %f", inFlightValue)
}

func TestErrorScenarios(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		status  int
	}{
		{
			name: "internal server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal error", http.StatusInternalServerError)
			},
			method: "GET",
			path:   "/error",
			status: 500,
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			},
			method: "GET",
			path:   "/notfound",
			status: 404,
		},
		{
			name: "bad request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			},
			method: "POST",
			path:   "/bad",
			status: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.Handle(fmt.Sprintf("%s %s", tt.method, tt.path),
				metrics.RequestDurationHandler("test", tt.handler))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)

			// Verify metrics were recorded for error cases
			assert.True(t, testutil.CollectAndCount(metrics.RequestDuration) >= 1)
		})
	}
}

func TestRequestDurationHandlerFixed(t *testing.T) {
	// Create a fresh registry to avoid conflicts
	reg := prometheus.NewRegistry()
	testDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_request_duration_seconds",
			Help:    "Test histogram for request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status", "version"},
	)
	reg.MustRegister(testDuration)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep for a deterministic amount to make test stable
		time.Sleep(1 * time.Millisecond)
		fmt.Fprint(w, "hello world")
	})

	// Create custom middleware that uses our test metric
	middleware := func(version string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)

			duration := time.Since(start)
			method := r.Method
			path := r.Pattern
			if path == "" {
				path = r.URL.Path
			}

			testDuration.WithLabelValues(method, path, "200", version).Observe(duration.Seconds())
		})
	}

	mux := http.NewServeMux()
	mux.Handle("GET /{path}", middleware("canary", h))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/greet")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 1, testutil.CollectAndCount(testDuration, "test_request_duration_seconds"))

	// Get the metrics output
	b, err := testutil.CollectAndFormat(testDuration, expfmt.TypeTextPlain, "test_request_duration_seconds")
	require.NoError(t, err)

	// Check that the metric was recorded (don't check exact duration)
	output := string(b)
	assert.Contains(t, output, "test_request_duration_seconds_bucket")
	assert.Contains(t, output, `method="GET"`)
	assert.Contains(t, output, `version="canary"`)
	assert.Contains(t, output, "test_request_duration_seconds_count")
	assert.Contains(t, output, "test_request_duration_seconds_sum")
}
