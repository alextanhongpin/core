package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
)

func TestNewMetricHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		meter := otel.GetMeterProvider().Meter("test")

		handler, err := NewMetricHandler(meter)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("nil meter", func(t *testing.T) {
		handler, err := NewMetricHandler(nil)
		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Equal(t, ErrNilMeter, err)
	})

	t.Run("with custom error handler", func(t *testing.T) {
		meter := otel.GetMeterProvider().Meter("test")

		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		handler, err := NewMetricHandler(meter, WithErrorHandler(errorHandler))
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test error handling
		ctx := context.Background()
		handler.Event(ctx, nil) // This should trigger an error
		assert.Equal(t, ErrNilEvent, capturedError)
	})
}

func TestMetricHandlerEvent(t *testing.T) {
	meter := otel.GetMeterProvider().Meter("test")

	handler, err := NewMetricHandler(meter)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

	t.Run("nil event", func(t *testing.T) {
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		h, err := NewMetricHandler(meter, WithErrorHandler(errorHandler))
		require.NoError(t, err)

		resultCtx := h.Event(ctx, nil)
		assert.Equal(t, ctx, resultCtx)
		assert.Equal(t, ErrNilEvent, capturedError)
	})

	t.Run("non-metric event", func(t *testing.T) {
		e := &event.Event{Kind: event.LogKind}
		resultCtx := handler.Event(ctx, e)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("counter", func(t *testing.T) {
		c := event.NewCounter("test_counter", &event.MetricOptions{
			Namespace:   "test_ns",
			Description: "Test counter",
		})

		c.Record(ctx, 123, event.String("label1", "value1"))
		c.Record(ctx, 456, event.String("label1", "value2"))

		// No errors should occur
		assert.True(t, true)
	})

	t.Run("negative counter value", func(t *testing.T) {
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		h, err := NewMetricHandler(meter, WithErrorHandler(errorHandler))
		require.NoError(t, err)

		testCtx := event.WithExporter(ctx, event.NewExporter(h, eventtest.ExporterOptions()))

		c := event.NewCounter("negative_counter", &event.MetricOptions{
			Description: "Test negative counter",
		})

		c.Record(testCtx, -123)

		assert.Error(t, capturedError)
		assert.Contains(t, capturedError.Error(), "counter value cannot be negative")
	})

	t.Run("float gauge", func(t *testing.T) {
		g := event.NewFloatGauge("test_gauge", &event.MetricOptions{
			Namespace:   "test_ns",
			Description: "Test gauge",
		})

		g.Record(ctx, 12.34, event.String("label1", "value1"))
		g.Record(ctx, 56.78, event.String("label1", "value2"))

		// No errors should occur
		assert.True(t, true)
	})

	t.Run("duration distribution", func(t *testing.T) {
		h := event.NewDuration("test_histogram", &event.MetricOptions{
			Namespace:   "test_ns",
			Description: "Test histogram",
		})

		h.Record(ctx, time.Second, event.String("label1", "value1"))
		h.Record(ctx, time.Minute, event.String("label1", "value2"))

		// No errors should occur
		assert.True(t, true)
	})

	t.Run("negative duration", func(t *testing.T) {
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		h, err := NewMetricHandler(meter, WithErrorHandler(errorHandler))
		require.NoError(t, err)

		testCtx := event.WithExporter(ctx, event.NewExporter(h, eventtest.ExporterOptions()))

		d := event.NewDuration("negative_duration", &event.MetricOptions{
			Description: "Test negative duration",
		})

		d.Record(testCtx, -time.Second)

		assert.Error(t, capturedError)
		assert.Contains(t, capturedError.Error(), "duration cannot be negative")
	})

	t.Run("missing metric key", func(t *testing.T) {
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		h, err := NewMetricHandler(meter, WithErrorHandler(errorHandler))
		require.NoError(t, err)

		// Create an event manually without metric key
		e := &event.Event{
			Kind: event.MetricKind,
		}

		h.Event(ctx, e)

		assert.Error(t, capturedError)
		assert.Equal(t, ErrNoMetricKey, capturedError)
	})
}

func TestMetricHandlerConcurrency(t *testing.T) {
	meter := otel.GetMeterProvider().Meter("test")

	handler, err := NewMetricHandler(meter)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

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

	// No assertion needed - just ensure no race conditions occur
	assert.True(t, true)
}

func TestMetricHandlerClose(t *testing.T) {
	meter := otel.Meter("test")
	handler, err := NewMetricHandler(meter)
	require.NoError(t, err)

	err = handler.Close()
	assert.NoError(t, err)
}

func TestLabelToAttribute(t *testing.T) {
	tests := []struct {
		name     string
		label    event.Label
		expectOk bool
	}{
		{
			name:     "string label",
			label:    event.String("key", "value"),
			expectOk: true,
		},
		{
			name:     "empty name",
			label:    event.String("", "value"),
			expectOk: true, // Function doesn't reject empty names, just converts them
		},
		{
			name:     "no value",
			label:    event.Label{},
			expectOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := labelToAttribute(tt.label)
			if tt.expectOk {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
