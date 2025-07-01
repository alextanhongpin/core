package metrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

// resetGlobalMetrics clears global metric state for isolation
func resetGlobalMetrics() {
	// Unregister all global metrics to prevent state pollution
	prometheus.DefaultRegisterer.Unregister(metrics.InFlightGauge)
	prometheus.DefaultRegisterer.Unregister(metrics.ResponseSize)
	prometheus.DefaultRegisterer.Unregister(metrics.RequestDuration)
	prometheus.DefaultRegisterer.Unregister(metrics.RED)

	// Re-register fresh instances
	metrics.InFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	metrics.ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		}, []string{})

	metrics.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "request_duration_seconds",
			Help: "A histogram of latencies for requests.",
		},
		[]string{"method", "path", "status", "version"},
	)

	metrics.RED = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "red",
			Help: "RED metrics",
		},
		[]string{"service", "action", "status"},
	)
}

func TestInFlightGauge(t *testing.T) {
	resetGlobalMetrics()
	prometheus.MustRegister(metrics.InFlightGauge)

	metrics.InFlightGauge.Inc()
	metrics.InFlightGauge.Add(2)

	is := assert.New(t)
	is.Equal(1, testutil.CollectAndCount(metrics.InFlightGauge, "in_flight_requests"))

	b, err := testutil.CollectAndFormat(metrics.InFlightGauge, expfmt.TypeTextPlain, "in_flight_requests")
	is.Nil(err)
	want := `# HELP in_flight_requests A gauge of requests currently being served by the wrapped handler.
# TYPE in_flight_requests gauge
in_flight_requests 3
`
	is.Equal(want, string(b))
}

func TestResponseSize(t *testing.T) {
	resetGlobalMetrics()
	prometheus.MustRegister(metrics.ResponseSize)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_ = metrics.ObserveResponseSize(r)

	is := assert.New(t)
	is.Equal(1, testutil.CollectAndCount(metrics.ResponseSize, "response_size_bytes"))

	b, err := testutil.CollectAndFormat(metrics.ResponseSize, expfmt.TypeTextPlain, "response_size_bytes")
	is.Nil(err)
	want := `# HELP response_size_bytes A histogram of response sizes for requests.
# TYPE response_size_bytes histogram
response_size_bytes_bucket{le="200"} 1
response_size_bytes_bucket{le="500"} 1
response_size_bytes_bucket{le="900"} 1
response_size_bytes_bucket{le="1500"} 1
response_size_bytes_bucket{le="+Inf"} 1
response_size_bytes_sum 23
response_size_bytes_count 1
`
	is.Equal(want, string(b))
}

func TestRequestDurationHandler(t *testing.T) {
	resetGlobalMetrics()
	prometheus.MustRegister(metrics.RequestDuration)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})
	mux := http.NewServeMux()
	mux.Handle("GET /{path}", metrics.RequestDurationHandler("canary", h))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/greet")
	is := assert.New(t)
	is.Nil(err)

	defer resp.Body.Close()

	is.Equal(1, testutil.CollectAndCount(metrics.RequestDuration, "request_duration_seconds"))

	b, err := testutil.CollectAndFormat(metrics.RequestDuration, expfmt.TypeTextPlain, "request_duration_seconds")
	is.Nil(err)

	// Instead of checking exact duration (which is flaky), check structure
	output := string(b)
	is.Contains(output, "# HELP request_duration_seconds A histogram of latencies for requests.")
	is.Contains(output, "# TYPE request_duration_seconds histogram")
	is.Contains(output, `method="GET"`)
	is.Contains(output, `path="/{path}"`)
	is.Contains(output, `status="200"`)
	is.Contains(output, `version="canary"`)
	is.Contains(output, "request_duration_seconds_count")
	is.Contains(output, "request_duration_seconds_sum")

	// Verify all buckets are present
	buckets := []string{"0.005", "0.01", "0.025", "0.05", "0.1", "0.25", "0.5", "1", "2.5", "5", "10", "+Inf"}
	for _, bucket := range buckets {
		is.Contains(output, fmt.Sprintf(`le="%s"`, bucket))
	}
}

func TestRED(t *testing.T) {
	resetGlobalMetrics()
	prometheus.MustRegister(metrics.RED)

	{
		red := metrics.NewRED("user_service", "login")
		red.Done()
	}

	{
		red := metrics.NewRED("user_service", "login")
		red.Fail()
		red.Done()
	}

	is := assert.New(t)
	is.Equal(2, testutil.CollectAndCount(metrics.RED, "red"))

	b, err := testutil.CollectAndFormat(metrics.RED, expfmt.TypeTextPlain, "red")
	is.Nil(err)
	want := `# HELP red RED metrics
# TYPE red histogram
red_bucket{action="login",service="user_service",status="err",le="0.005"} 1
red_bucket{action="login",service="user_service",status="err",le="0.01"} 1
red_bucket{action="login",service="user_service",status="err",le="0.025"} 1
red_bucket{action="login",service="user_service",status="err",le="0.05"} 1
red_bucket{action="login",service="user_service",status="err",le="0.1"} 1
red_bucket{action="login",service="user_service",status="err",le="0.25"} 1
red_bucket{action="login",service="user_service",status="err",le="0.5"} 1
red_bucket{action="login",service="user_service",status="err",le="1"} 1
red_bucket{action="login",service="user_service",status="err",le="2.5"} 1
red_bucket{action="login",service="user_service",status="err",le="5"} 1
red_bucket{action="login",service="user_service",status="err",le="10"} 1
red_bucket{action="login",service="user_service",status="err",le="+Inf"} 1
red_sum{action="login",service="user_service",status="err"} 0
red_count{action="login",service="user_service",status="err"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.005"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.01"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.025"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.05"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.1"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.25"} 1
red_bucket{action="login",service="user_service",status="ok",le="0.5"} 1
red_bucket{action="login",service="user_service",status="ok",le="1"} 1
red_bucket{action="login",service="user_service",status="ok",le="2.5"} 1
red_bucket{action="login",service="user_service",status="ok",le="5"} 1
red_bucket{action="login",service="user_service",status="ok",le="10"} 1
red_bucket{action="login",service="user_service",status="ok",le="+Inf"} 1
red_sum{action="login",service="user_service",status="ok"} 0
red_count{action="login",service="user_service",status="ok"} 1
`
	is.Equal(want, string(b))
}
