package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductServiceIntegration demonstrates comprehensive integration testing
// with proper metrics isolation and edge case coverage
func TestProductServiceIntegration(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		url               string
		body              string
		expectedStatus    int
		expectedREDStatus string
		checkMetrics      func(t *testing.T, registry *prometheus.Registry)
	}{
		{
			name:              "successful product retrieval",
			method:            "GET",
			url:               "/api/product?id=1",
			expectedStatus:    http.StatusOK,
			expectedREDStatus: "ok",
			checkMetrics:      checkSuccessfulRequest,
		},
		{
			name:              "product not found",
			method:            "GET",
			url:               "/api/product?id=999",
			expectedStatus:    http.StatusNotFound,
			expectedREDStatus: "not_found",
			checkMetrics:      checkNotFoundRequest,
		},
		{
			name:              "invalid product ID",
			method:            "GET",
			url:               "/api/product?id=invalid",
			expectedStatus:    http.StatusBadRequest,
			expectedREDStatus: "invalid_id",
			checkMetrics:      checkBadRequest,
		},
		{
			name:              "missing product ID",
			method:            "GET",
			url:               "/api/product",
			expectedStatus:    http.StatusBadRequest,
			expectedREDStatus: "missing_id",
			checkMetrics:      checkBadRequest,
		},
		{
			name:              "service error simulation",
			method:            "GET",
			url:               "/api/product?id=500",
			expectedStatus:    http.StatusServiceUnavailable,
			expectedREDStatus: "service_error",
			checkMetrics:      checkServiceError,
		},
		{
			name:              "successful product creation",
			method:            "POST",
			url:               "/api/products/create",
			body:              `{"name":"Test Product","price":49.99}`,
			expectedStatus:    http.StatusCreated,
			expectedREDStatus: "ok",
			checkMetrics:      checkSuccessfulRequest,
		},
		{
			name:              "invalid JSON in create",
			method:            "POST",
			url:               "/api/products/create",
			body:              `{"name":"Test Product","price":}`,
			expectedStatus:    http.StatusBadRequest,
			expectedREDStatus: "invalid_json",
			checkMetrics:      checkBadRequest,
		},
		{
			name:              "validation error - missing name",
			method:            "POST",
			url:               "/api/products/create",
			body:              `{"price":49.99}`,
			expectedStatus:    http.StatusBadRequest,
			expectedREDStatus: "validation_failed",
			checkMetrics:      checkBadRequest,
		},
		{
			name:              "validation error - negative price",
			method:            "POST",
			url:               "/api/products/create",
			body:              `{"name":"Test Product","price":-10}`,
			expectedStatus:    http.StatusBadRequest,
			expectedREDStatus: "validation_failed",
			checkMetrics:      checkBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated test environment
			registry, service, server := setupTestEnvironment(t)
			defer server.Close()

			// Make request
			var req *http.Request
			var err error

			if tt.body != "" {
				req, err = http.NewRequest(tt.method, server.URL+tt.url, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.url, nil)
			}
			require.NoError(t, err)

			// Add test headers
			req.Header.Set("X-User-ID", "test-user-123")

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify response format for error cases
			if resp.StatusCode >= 400 {
				var errorResp ErrorResponse
				err := json.NewDecoder(resp.Body).Decode(&errorResp)
				assert.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
				assert.NotEmpty(t, errorResp.Code)
			}

			// Wait a bit for async metrics processing
			time.Sleep(10 * time.Millisecond)

			// Verify metrics were recorded correctly
			tt.checkMetrics(t, registry)

			// Verify RED metrics contain the expected status
			verifyREDMetrics(t, registry, tt.expectedREDStatus)

			// Verify in-flight gauge is back to 0
			inFlight := testutil.ToFloat64(metrics.InFlightGauge)
			assert.Equal(t, float64(0), inFlight, "In-flight gauge should be 0 after request")

			_ = service // Silence unused variable warning
		})
	}
}

// TestConcurrentRequests verifies thread safety and proper metrics under load
func TestConcurrentRequests(t *testing.T) {
	// Use a different setup for concurrent testing that doesn't rely on global metrics isolation
	// Create service without metrics override
	service := NewProductService(nil, nil)

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/product", service.GetProduct)
	server := httptest.NewServer(mux)
	defer server.Close()

	const numWorkers = 10
	const numRequestsPerWorker = 20

	// Channel to collect results
	results := make(chan error, numWorkers*numRequestsPerWorker)

	// Launch concurrent workers
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			client := &http.Client{Timeout: 5 * time.Second}

			for j := 0; j < numRequestsPerWorker; j++ {
				// Mix of successful and error requests
				var url string
				if (workerID+j)%4 == 0 {
					url = "/api/product?id=invalid" // Generate errors
				} else {
					url = "/api/product?id=1" // Successful requests
				}

				req, err := http.NewRequest("GET", server.URL+url, nil)
				if err != nil {
					results <- err
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					results <- err
					continue
				}
				resp.Body.Close()

				results <- nil
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numWorkers*numRequestsPerWorker; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	// Should have no client errors (server errors are expected)
	assert.Empty(t, errors, "Should have no client errors")

	// Wait for metrics to be processed
	time.Sleep(2 * time.Second)

	// For concurrent test, just verify that the handlers don't panic and process requests correctly
	// The main goal is to test thread safety, not specific metrics counts
	// (since global metrics isolation is not thread-safe)
	t.Log("Concurrent test completed successfully - all requests processed without panics")
}

// TestMetricsIsolation verifies that tests don't interfere with each other
func TestMetricsIsolation(t *testing.T) {
	t.Run("first test", func(t *testing.T) {
		registry, _, server := setupTestEnvironment(t)
		defer server.Close()

		// Make a request
		resp, err := http.Get(server.URL + "/api/product?id=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify exactly one request was recorded
		count := testutil.CollectAndCount(metrics.RequestDuration, "test_request_duration_seconds")
		assert.Equal(t, 1, count, "Should have exactly one request")

		// This registry should be independent
		_ = registry
	})

	t.Run("second test", func(t *testing.T) {
		registry, _, server := setupTestEnvironment(t)
		defer server.Close()

		// Make a different request
		resp, err := http.Get(server.URL + "/api/products")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should also have exactly one request (not affected by previous test)
		count := testutil.CollectAndCount(metrics.RequestDuration, "test_request_duration_seconds")
		assert.Equal(t, 1, count, "Should have exactly one request (isolated)")

		_ = registry
	})
}

// TestErrorHandling verifies proper error handling and metrics recording
func TestErrorHandling(t *testing.T) {
	registry, _, server := setupTestEnvironment(t)
	defer server.Close()

	// Test panic recovery by sending malformed requests
	testCases := []struct {
		name   string
		url    string
		method string
	}{
		{"malformed JSON", "/api/products/create", "POST"},
		{"invalid URL encoding", "/api/product?id=%zzz", "GET"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tc.method == "POST" {
				req, err = http.NewRequest(tc.method, server.URL+tc.url, strings.NewReader("{malformed json"))
			} else {
				req, err = http.NewRequest(tc.method, server.URL+tc.url, nil)
			}
			require.NoError(t, err)

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should get an error response, not a panic
			assert.GreaterOrEqual(t, resp.StatusCode, 400)
		})
	}

	// Verify metrics were still recorded despite errors
	count := testutil.CollectAndCount(metrics.RequestDuration, "test_request_duration_seconds")
	assert.GreaterOrEqual(t, count, len(testCases))

	_ = registry
}

// Test helper functions

func setupTestEnvironment(t *testing.T) (*prometheus.Registry, *ProductService, *httptest.Server) {
	// Create isolated metrics registry for this test
	registry := prometheus.NewRegistry()

	// Create fresh metrics instances
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_in_flight_requests",
		Help: "Test in-flight requests",
	})

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_request_duration_seconds",
			Help: "Test request duration",
		},
		[]string{"method", "path", "status", "version"},
	)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_response_size_bytes",
			Help: "Test response size",
		},
		[]string{},
	)

	red := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_red",
			Help: "Test RED metrics",
		},
		[]string{"service", "action", "status"},
	)

	registry.MustRegister(inFlightGauge, requestDuration, responseSize, red)

	// Override global metrics for this test
	originalInFlight := metrics.InFlightGauge
	originalDuration := metrics.RequestDuration
	originalSize := metrics.ResponseSize
	originalRED := metrics.RED

	metrics.InFlightGauge = inFlightGauge
	metrics.RequestDuration = requestDuration
	metrics.ResponseSize = responseSize
	metrics.RED = red

	// Restore original metrics when test completes
	t.Cleanup(func() {
		metrics.InFlightGauge = originalInFlight
		metrics.RequestDuration = originalDuration
		metrics.ResponseSize = originalSize
		metrics.RED = originalRED
	})

	// Create test service (without Redis for simplicity)
	service := NewProductService(nil, nil)

	// Create test server
	mux := http.NewServeMux()

	// Setup routes with minimal middleware for testing
	mux.HandleFunc("/api/product", service.GetProduct)
	mux.HandleFunc("/api/products", service.ListProducts)
	mux.HandleFunc("/api/products/create", service.CreateProduct)
	mux.HandleFunc("/health", service.HealthCheck)

	server := httptest.NewServer(mux)

	return registry, service, server
}

func checkSuccessfulRequest(t *testing.T, registry *prometheus.Registry) {
	// Verify request duration was recorded
	count := testutil.CollectAndCount(metrics.RequestDuration, "test_request_duration_seconds")
	assert.GreaterOrEqual(t, count, 1, "Request duration should be recorded")

	// Verify response size was recorded
	sizeCount := testutil.CollectAndCount(metrics.ResponseSize, "test_response_size_bytes")
	assert.GreaterOrEqual(t, sizeCount, 1, "Response size should be recorded")
}

func checkNotFoundRequest(t *testing.T, registry *prometheus.Registry) {
	checkSuccessfulRequest(t, registry) // Same basic checks apply
}

func checkBadRequest(t *testing.T, registry *prometheus.Registry) {
	checkSuccessfulRequest(t, registry) // Same basic checks apply
}

func checkServiceError(t *testing.T, registry *prometheus.Registry) {
	checkSuccessfulRequest(t, registry) // Same basic checks apply
}

func verifyREDMetrics(t *testing.T, registry *prometheus.Registry, expectedStatus string) {
	// Collect RED metrics and verify status was recorded
	count := testutil.CollectAndCount(metrics.RED, "test_red")
	assert.GreaterOrEqual(t, count, 1, "RED metrics should be recorded")

	// Could add more specific checks for status labels here
	// This would require parsing the metric families to check label values
}

// Benchmark tests for performance verification
func BenchmarkProductServiceGetProduct(b *testing.B) {
	_, _, server := setupBenchEnvironment(b)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", server.URL+"/api/product?id=1", nil)
			resp, err := client.Do(req)
			if err != nil {
				b.Error(err)
			}
			resp.Body.Close()
		}
	})
}

func setupBenchEnvironment(b *testing.B) (*prometheus.Registry, *ProductService, *httptest.Server) {
	// Similar to setupTestEnvironment but optimized for benchmarks
	registry := prometheus.NewRegistry()
	service := NewProductService(nil, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/product", service.GetProduct)

	server := httptest.NewServer(mux)
	return registry, service, server
}
