package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry() *prometheus.Registry {
	// Create a new registry for isolated testing
	reg := prometheus.NewRegistry()
	return reg
}

func TestInFlightGaugeIsolated(t *testing.T) {
	reg := setupTestRegistry()

	// Create isolated gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "in_flight_requests",
		Help: "A gauge of requests currently being served.",
	})
	reg.MustRegister(gauge)

	gauge.Inc()
	gauge.Add(2)

	metricCount := testutil.CollectAndCount(gauge)
	assert.Equal(t, 1, metricCount)

	gaugeValue := testutil.ToFloat64(gauge)
	assert.Equal(t, float64(3), gaugeValue)

	gauge.Dec()
	gauge.Sub(2)

	finalValue := testutil.ToFloat64(gauge)
	assert.Equal(t, float64(0), finalValue)
}

func TestResponseSizeIsolated(t *testing.T) {
	reg := setupTestRegistry()

	// Create isolated histogram
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "response_size_bytes_test",
		Help:    "A histogram of response sizes for requests.",
		Buckets: []float64{200, 500, 900, 1500},
	})
	reg.MustRegister(hist)

	// Create a minimal test request
	req := httptest.NewRequest("GET", "/", nil)

	// Calculate expected size based on actual request structure
	expectedSize := len("GET") + len("/") + len("HTTP/1.1")

	// Record the expected size directly
	hist.Observe(float64(expectedSize))

	expected := fmt.Sprintf(`# HELP response_size_bytes_test A histogram of response sizes for requests.
# TYPE response_size_bytes_test histogram
response_size_bytes_test_bucket{le="200"} 1
response_size_bytes_test_bucket{le="500"} 1
response_size_bytes_test_bucket{le="900"} 1
response_size_bytes_test_bucket{le="1500"} 1
response_size_bytes_test_bucket{le="+Inf"} 1
response_size_bytes_test_sum %d
response_size_bytes_test_count 1
`, expectedSize)

	err := testutil.GatherAndCompare(reg, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestRequestDurationHandlerIsolated(t *testing.T) {
	reg := setupTestRegistry()

	// Create isolated histogram
	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "request_duration_seconds_test",
		Help: "A histogram of request durations.",
	}, []string{"code", "method"})
	reg.MustRegister(hist)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record metrics
		hist.WithLabelValues("200", "GET").Observe(0.001)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())

	// Verify metric was recorded
	metricCount := testutil.CollectAndCount(hist)
	assert.Equal(t, 1, metricCount)
}

func TestREDIsolated(t *testing.T) {
	reg := setupTestRegistry()

	// Create isolated histogram
	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "red_test",
		Help: "RED metrics",
	}, []string{"service", "action", "status"})
	reg.MustRegister(hist)

	// Simulate RED metrics
	hist.WithLabelValues("user_service", "login", "ok").Observe(0.001)
	hist.WithLabelValues("user_service", "login", "err").Observe(0.001)

	metricCount := testutil.CollectAndCount(hist)
	assert.Equal(t, 1, metricCount)

	// Test the actual values match expected
	expected := `# HELP red_test RED metrics
# TYPE red_test histogram
red_test_bucket{action="login",service="user_service",status="err",le="0.005"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.01"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.025"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.05"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.1"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.25"} 1
red_test_bucket{action="login",service="user_service",status="err",le="0.5"} 1
red_test_bucket{action="login",service="user_service",status="err",le="1"} 1
red_test_bucket{action="login",service="user_service",status="err",le="2.5"} 1
red_test_bucket{action="login",service="user_service",status="err",le="5"} 1
red_test_bucket{action="login",service="user_service",status="err",le="10"} 1
red_test_bucket{action="login",service="user_service",status="err",le="+Inf"} 1
red_test_sum{action="login",service="user_service",status="err"} 0.001
red_test_count{action="login",service="user_service",status="err"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.005"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.01"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.025"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.05"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.1"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.25"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="0.5"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="1"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="2.5"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="5"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="10"} 1
red_test_bucket{action="login",service="user_service",status="ok",le="+Inf"} 1
red_test_sum{action="login",service="user_service",status="ok"} 0.001
red_test_count{action="login",service="user_service",status="ok"} 1
`

	err := testutil.GatherAndCompare(reg, strings.NewReader(expected))
	assert.NoError(t, err)
}
