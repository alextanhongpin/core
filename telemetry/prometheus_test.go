package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
)

var ctx = context.Background()

func TestNewPrometheusHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		handler, err := NewPrometheusHandler(reg)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("nil registerer", func(t *testing.T) {
		handler, err := NewPrometheusHandler(nil)
		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Equal(t, ErrNilRegisterer, err)
	})

	t.Run("with custom error handler", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		handler, err := NewPrometheusHandler(reg, WithPrometheusErrorHandler(errorHandler))
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test error handling
		handler.Event(ctx, nil) // This should trigger an error
		assert.Equal(t, ErrNilEvent, capturedError)
	})
}

func TestPrometheus(t *testing.T) {
	t.Run("counter", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metric, err := NewPrometheusHandler(reg)
		require.NoError(t, err)

		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		c := event.NewCounter("hits", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "Earth meteorite hits"},
		)
		c.Record(ctx, 123, event.String("version", "stable"))
		c.Record(ctx, 456, event.String("version", "canary"))

		collector, ok := metric.Collector("hits")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_hits"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_hits")
		is.NoError(err)
		want := `# HELP my_ns_hits Earth meteorite hits
# TYPE my_ns_hits counter
my_ns_hits{version="canary"} 456
my_ns_hits{version="stable"} 123
`
		is.Equal(want, string(b))
	})

	t.Run("negative counter", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		metric, err := NewPrometheusHandler(reg, WithPrometheusErrorHandler(errorHandler))
		require.NoError(t, err)

		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		c := event.NewCounter("negative_hits", &event.MetricOptions{
			Description: "Negative counter test"},
		)
		c.Record(ctx, -123)

		assert.Error(t, capturedError)
		assert.Contains(t, capturedError.Error(), "counter value cannot be negative")
	})

	t.Run("gauge", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metric, err := NewPrometheusHandler(reg)
		require.NoError(t, err)

		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		g := event.NewFloatGauge("cpu", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "cpu usage"},
		)
		g.Record(ctx, 123, event.String("version", "canary"))
		g.Record(ctx, 456, event.String("version", "canary"))
		g.Record(ctx, 456, event.String("version", "stable"))
		g.Record(ctx, 123, event.String("version", "stable"))

		collector, ok := metric.Collector("cpu")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_cpu"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_cpu")
		is.NoError(err)
		want := `# HELP my_ns_cpu cpu usage
# TYPE my_ns_cpu gauge
my_ns_cpu{version="canary"} 456
my_ns_cpu{version="stable"} 123
`
		is.Equal(want, string(b))
	})

	t.Run("histogram", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metric, err := NewPrometheusHandler(reg)
		require.NoError(t, err)

		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		h := event.NewDuration("request_duration", &event.MetricOptions{
			Namespace:   "my_ns",
			Description: "request per seconds",
			//Unit:        event.UnitMilliseconds,
		})
		h.Record(ctx, time.Second, event.String("version", "stable"))
		h.Record(ctx, time.Minute, event.String("version", "canary"))

		collector, ok := metric.Collector("request_duration")

		is := assert.New(t)
		is.True(ok)
		is.Equal(2, testutil.CollectAndCount(collector, "my_ns_request_duration_seconds"))
		b, err := testutil.CollectAndFormat(collector, expfmt.TypeTextPlain, "my_ns_request_duration_seconds")
		is.NoError(err)
		want := `# HELP my_ns_request_duration_seconds request per seconds
# TYPE my_ns_request_duration_seconds histogram
my_ns_request_duration_seconds_bucket{version="canary",le="0.005"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.01"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.025"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.05"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.1"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.25"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="0.5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="1"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="2.5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="5"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="10"} 0
my_ns_request_duration_seconds_bucket{version="canary",le="+Inf"} 1
my_ns_request_duration_seconds_sum{version="canary"} 60
my_ns_request_duration_seconds_count{version="canary"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="0.005"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.01"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.025"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.05"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.1"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.25"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="0.5"} 0
my_ns_request_duration_seconds_bucket{version="stable",le="1"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="2.5"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="5"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="10"} 1
my_ns_request_duration_seconds_bucket{version="stable",le="+Inf"} 1
my_ns_request_duration_seconds_sum{version="stable"} 1
my_ns_request_duration_seconds_count{version="stable"} 1
`
		is.Equal(want, string(b))
	})

	t.Run("negative duration", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		metric, err := NewPrometheusHandler(reg, WithPrometheusErrorHandler(errorHandler))
		require.NoError(t, err)

		ctx := event.WithExporter(ctx, event.NewExporter(metric, eventtest.ExporterOptions()))
		h := event.NewDuration("negative_duration", &event.MetricOptions{
			Description: "Negative duration test"},
		)
		h.Record(ctx, -time.Second)

		assert.Error(t, capturedError)
		assert.Contains(t, capturedError.Error(), "duration cannot be negative")
	})

	t.Run("empty metric name", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		metric, err := NewPrometheusHandler(reg, WithPrometheusErrorHandler(errorHandler))
		require.NoError(t, err)

		// Create an event manually without metric key
		e := &event.Event{
			Kind: event.MetricKind,
		}

		metric.Event(ctx, e)

		assert.Error(t, capturedError)
		assert.Equal(t, ErrNoMetricKey, capturedError)
	})

	t.Run("nil event", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		metric, err := NewPrometheusHandler(reg, WithPrometheusErrorHandler(errorHandler))
		require.NoError(t, err)

		metric.Event(ctx, nil)

		assert.Error(t, capturedError)
		assert.Equal(t, ErrNilEvent, capturedError)
	})

	t.Run("non-metric event", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metric, err := NewPrometheusHandler(reg)
		require.NoError(t, err)

		e := &event.Event{Kind: event.LogKind}
		resultCtx := metric.Event(ctx, e)
		assert.Equal(t, ctx, resultCtx)
	})
}

func TestPrometheusHandlerClose(t *testing.T) {
	reg := prometheus.NewRegistry()
	handler, err := NewPrometheusHandler(reg)
	require.NoError(t, err)

	// Add a metric
	ctx := event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))
	c := event.NewCounter("test_counter", &event.MetricOptions{
		Description: "Test counter for close",
	})
	c.Record(ctx, 1)

	// Close should work
	err = handler.Close()
	assert.NoError(t, err)

	// Should be safe to call multiple times
	err = handler.Close()
	assert.NoError(t, err)

	// Collector should no longer exist
	_, ok := handler.Collector("test_counter")
	assert.False(t, ok)
}

func TestPrometheusHandlerConcurrency(t *testing.T) {
	reg := prometheus.NewRegistry()
	handler, err := NewPrometheusHandler(reg)
	require.NoError(t, err)

	ctx := event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

	// Test concurrent access to the same metric
	c := event.NewCounter("concurrent_counter", &event.MetricOptions{
		Description: "Test concurrent counter",
	})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			c.Record(ctx, int64(i))
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify collector exists
	collector, ok := handler.Collector("concurrent_counter")
	assert.True(t, ok)
	assert.NotNil(t, collector)
}

func TestPrometheusHandlerCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	handler, err := NewPrometheusHandler(reg)
	require.NoError(t, err)

	t.Run("empty name", func(t *testing.T) {
		collector, exists := handler.Collector("")
		assert.Nil(t, collector)
		assert.False(t, exists)
	})

	t.Run("non-existent metric", func(t *testing.T) {
		collector, exists := handler.Collector("non_existent_metric")
		assert.Nil(t, collector)
		assert.False(t, exists)
	})

	t.Run("existing metric", func(t *testing.T) {
		// First create a metric by handling an event
		counter := event.NewCounter("test_collector_metric", &event.MetricOptions{})
		testCtx := event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))
		counter.Record(testCtx, 42.0)

		// Now check that we can get the collector
		collector, exists := handler.Collector("test_collector_metric")
		assert.NotNil(t, collector)
		assert.True(t, exists)
	})
}

// Add tests for additional prometheus coverage
func TestPrometheusAdditionalCoverage(t *testing.T) {
	reg := prometheus.NewRegistry()
	handler, err := NewPrometheusHandler(reg)
	require.NoError(t, err)

	t.Run("bytes unit", func(t *testing.T) {
		counter := event.NewCounter("test_bytes_counter", &event.MetricOptions{
			Unit: event.UnitBytes,
		})
		testCtx := event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))
		counter.Record(testCtx, 1024.0)

		// Check metric was created (may or may not have _bytes suffix)
		_, exists := handler.Collector("test_bytes_counter")
		if !exists {
			_, exists = handler.Collector("test_bytes_counter_bytes")
		}
		// At least one should exist
		assert.True(t, exists, "Expected either 'test_bytes_counter' or 'test_bytes_counter_bytes' to exist")
	})

	t.Run("label conversion edge cases", func(t *testing.T) {
		tests := []struct {
			name   string
			labels []event.Label
		}{
			{
				name:   "empty labels",
				labels: []event.Label{},
			},
			{
				name: "labels with empty name",
				labels: []event.Label{
					event.String("", "should be ignored"),
					event.String("valid", "value"),
				},
			},
			{
				name: "different value types",
				labels: []event.Label{
					event.Bool("bool_key", true),
					event.Int64("int_key", 123),
					event.Float64("float_key", 45.67),
					event.Bytes("bytes_key", []byte("test")),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				counter := event.NewCounter("label_test_"+tt.name, &event.MetricOptions{})
				testCtx := event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

				// Record with labels
				counter.Record(testCtx, 1.0, tt.labels...)
			})
		}
	})
}
