package metrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// TestInFlightGaugeBasic tests the gauge functionality without global state pollution
func TestInFlightGaugeBasic(t *testing.T) {
	// Test basic functionality without relying on global state
	assert.NotNil(t, metrics.InFlightGauge)

	// Store initial value
	initial := testutil.ToFloat64(metrics.InFlightGauge)

	metrics.InFlightGauge.Inc()
	incremented := testutil.ToFloat64(metrics.InFlightGauge)
	assert.Equal(t, initial+1, incremented)

	metrics.InFlightGauge.Dec()
	decremented := testutil.ToFloat64(metrics.InFlightGauge)
	assert.Equal(t, initial, decremented)
}

func TestResponseSizeComputation(t *testing.T) {
	// Test the response size calculation logic
	testCases := []struct {
		name            string
		method          string
		url             string
		headers         map[string]string
		expectedMinSize int // minimum expected size
	}{
		{
			name:            "simple GET",
			method:          "GET",
			url:             "/",
			headers:         nil,
			expectedMinSize: 10, // at least method + path
		},
		{
			name:            "GET with headers",
			method:          "GET",
			url:             "/api/test",
			headers:         map[string]string{"Content-Type": "application/json"},
			expectedMinSize: 30, // method + path + header
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			size := metrics.ObserveResponseSize(req)
			assert.GreaterOrEqual(t, int(size), tc.expectedMinSize)
		})
	}
}

func TestRequestDurationHandlerBasic(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	instrumentedHandler := metrics.RequestDurationHandler("v1.0", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	instrumentedHandler.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestREDTrackerBasic(t *testing.T) {
	red := metrics.NewRED("test_service", "test_action")
	assert.NotNil(t, red)

	// Test that we can call methods without panicking
	red.SetStatus("processing")
	red.Done()

	// Test error state
	red2 := metrics.NewRED("test_service", "test_action_err")
	red2.Fail()
	red2.Done()
}

func TestREDTrackerWithCustomStatus(t *testing.T) {
	red := metrics.NewRED("custom_service", "custom_action")
	assert.NotNil(t, red)

	red.SetStatus("custom_error")
	red.Fail()
	red.Done()
}
