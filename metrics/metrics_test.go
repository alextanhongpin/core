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

func TestInFlightGauge(t *testing.T) {
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
	want := `# HELP request_duration_seconds A histogram of latencies for requests.
# TYPE request_duration_seconds histogram
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.005"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.01"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.025"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.05"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.1"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.25"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="0.5"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="1"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="2.5"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="5"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="10"} 1
request_duration_seconds_bucket{method="GET",path="/{path}",status="200",version="canary",le="+Inf"} 1
request_duration_seconds_sum{method="GET",path="/{path}",status="200",version="canary"} 0
request_duration_seconds_count{method="GET",path="/{path}",status="200",version="canary"} 1
`
	is.Equal(want, string(b))
}

func TestRED(t *testing.T) {
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
