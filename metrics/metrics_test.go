package metrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func reset() {
	metrics.InFlightGauge.Set(0)
	metrics.RequestDuration.Reset()
	metrics.ResponseSize.Reset()
	metrics.RED.Reset()
}

func TestRequestDurationHandler(t *testing.T) {
	reset()

	is := assert.New(t)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		is.Equal(1.0, testutil.ToFloat64(metrics.InFlightGauge))
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "hello world")
	})
	mux := http.NewServeMux()
	mux.Handle("GET /users/{id}", h)
	srv := httptest.NewServer(metrics.RequestDurationHandler("canary", mux))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/users/1")
	is.Nil(err)
	resp.Body.Close()
	is.Equal(http.StatusCreated, resp.StatusCode)

	// The raw URL path must not leak into the label.
	is.Equal(1, testutil.CollectAndCount(metrics.RequestDuration))
	is.Equal(uint64(1), histogramCount(t, metrics.RequestDuration.WithLabelValues("GET", "/users/{id}", "201", "canary")))
	is.Equal(0.0, testutil.ToFloat64(metrics.InFlightGauge))

	// The response size is the number of body bytes written, not the
	// request size.
	is.Equal(11.0, histogramSum(t, metrics.ResponseSize.WithLabelValues()))
}

func TestRequestDurationHandlerUnmatched(t *testing.T) {
	reset()

	srv := httptest.NewServer(metrics.RequestDurationHandler("v1", http.NewServeMux()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/random/path")
	is := assert.New(t)
	is.Nil(err)
	resp.Body.Close()

	is.Equal(uint64(1), histogramCount(t, metrics.RequestDuration.WithLabelValues("GET", "", "404", "v1")))
}

func TestRequestDurationHandlerPanic(t *testing.T) {
	reset()

	h := metrics.RequestDurationHandler("v1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	is := assert.New(t)
	is.PanicsWithValue("boom", func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	})
	is.Equal(uint64(1), histogramCount(t, metrics.RequestDuration.WithLabelValues("GET", "", "500", "v1")))
	is.Equal(0.0, testutil.ToFloat64(metrics.InFlightGauge))
}

func TestRequestDurationHandlerInformational(t *testing.T) {
	reset()

	var flushed bool
	h := metrics.RequestDurationHandler("v1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusAccepted)
		_, flushed = w.(http.Flusher)
		w.(http.Flusher).Flush()
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	is := assert.New(t)
	is.True(flushed)
	is.Equal(uint64(1), histogramCount(t, metrics.RequestDuration.WithLabelValues("GET", "", "202", "v1")))
}

func TestRED(t *testing.T) {
	reset()

	{
		red := metrics.NewRED("user_service", "login")
		time.Sleep(20 * time.Millisecond)
		red.Done()
	}

	{
		red := metrics.NewRED("user_service", "login")
		red.Fail()
		red.Done()
	}

	is := assert.New(t)
	is.Equal(2, testutil.CollectAndCount(metrics.RED, "red_duration_seconds"))

	ok := metrics.RED.WithLabelValues("user_service", "login", metrics.OK)
	is.Equal(uint64(1), histogramCount(t, ok))

	// Durations are observed in seconds, matching the buckets.
	sum := histogramSum(t, ok)
	is.GreaterOrEqual(sum, 0.02)
	is.Less(sum, 1.0)

	is.Equal(uint64(1), histogramCount(t, metrics.RED.WithLabelValues("user_service", "login", metrics.Err)))
}
