package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

func TestNewSlogHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		handler, err := NewSlogHandler(logger)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("nil logger", func(t *testing.T) {
		handler, err := NewSlogHandler(nil)
		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Equal(t, ErrNilLogger, err)
	})

	t.Run("with custom error handler", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		handler, err := NewSlogHandler(logger, WithSlogErrorHandler(errorHandler))
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test error handling
		ctx := context.Background()
		handler.Event(ctx, nil) // This should trigger an error
		assert.Equal(t, ErrNilEvent, capturedError)
	})
}

func TestSlogHandlerEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler, err := NewSlogHandler(logger)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

	t.Run("nil event", func(t *testing.T) {
		buf.Reset()

		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		h, err := NewSlogHandler(logger, WithSlogErrorHandler(errorHandler))
		require.NoError(t, err)

		resultCtx := h.Event(ctx, nil)
		assert.Equal(t, ctx, resultCtx)
		assert.Equal(t, ErrNilEvent, capturedError)
	})

	t.Run("non-log event", func(t *testing.T) {
		buf.Reset()

		e := &event.Event{Kind: event.MetricKind}
		resultCtx := handler.Event(ctx, e)
		assert.Equal(t, ctx, resultCtx)
		assert.Empty(t, buf.String())
	})

	t.Run("simple log", func(t *testing.T) {
		buf.Reset()

		event.Log(ctx, "test message", event.String("key", "value"))

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test message", logEntry["msg"])
		assert.Equal(t, "value", logEntry["key"])
		assert.Equal(t, "INFO", logEntry["level"])
	})

	t.Run("error log", func(t *testing.T) {
		buf.Reset()

		event.Log(ctx, "error occurred",
			event.String("error", "something went wrong"),
			event.String("component", "test"))

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "error occurred", logEntry["msg"])
		assert.Equal(t, "something went wrong", logEntry["error"])
		assert.Equal(t, "test", logEntry["component"])
		assert.Equal(t, "ERROR", logEntry["level"])
	})

	t.Run("log with source", func(t *testing.T) {
		buf.Reset()

		// Create an event with source information
		e := &event.Event{
			Kind: event.LogKind,
			Source: event.Source{
				Space: "myapp",
				Owner: "user",
				Name:  "function",
			},
			Labels: []event.Label{
				event.String("msg", "test with source"),
				event.String("key", "value"),
			},
		}

		handler.Event(ctx, e)

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test with source", logEntry["msg"])
		assert.Equal(t, "myapp", logEntry["in"])
		assert.Equal(t, "user", logEntry["owner"])
		assert.Equal(t, "function", logEntry["name"])
	})

	t.Run("log with various label types", func(t *testing.T) {
		buf.Reset()

		event.Log(ctx, "test labels",
			event.String("string_val", "text"),
			event.Int64("int_val", 42),
			event.Float64("float_val", 3.14),
			event.Bool("bool_val", true))

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test labels", logEntry["msg"])
		assert.Equal(t, "text", logEntry["string_val"])
		assert.Equal(t, float64(42), logEntry["int_val"])
		assert.Equal(t, 3.14, logEntry["float_val"])
		assert.Equal(t, true, logEntry["bool_val"])
	})

	t.Run("empty label values", func(t *testing.T) {
		buf.Reset()

		// Create an event with empty labels
		e := &event.Event{
			Kind: event.LogKind,
			Labels: []event.Label{
				event.String("msg", "test empty labels"),
				{Name: ""},            // Empty name
				{Name: "empty_value"}, // No value
			},
		}

		handler.Event(ctx, e)

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test empty labels", logEntry["msg"])
		// Empty labels should not appear
		_, hasEmptyName := logEntry[""]
		assert.False(t, hasEmptyName)
		_, hasEmptyValue := logEntry["empty_value"]
		assert.False(t, hasEmptyValue)
	})

	t.Run("missing message", func(t *testing.T) {
		buf.Reset()

		// Create an event without a message
		e := &event.Event{
			Kind: event.LogKind,
			Labels: []event.Label{
				event.String("key", "value"),
			},
		}

		handler.Event(ctx, e)

		assert.NotEmpty(t, buf.String())

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		// Should have empty message
		assert.Equal(t, "", logEntry["msg"])
		assert.Equal(t, "value", logEntry["key"])
	})
}

func TestSlogHandlerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler, err := NewSlogHandler(logger)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(handler, eventtest.ExporterOptions()))

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			event.Log(ctx, "concurrent log", event.Int64("id", int64(i)))
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have some log entries
	assert.NotEmpty(t, buf.String())
}

// Test to improve coverage for label function
func TestSlogLabelTypeCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler, err := NewSlogHandler(logger)
	require.NoError(t, err)

	tests := []struct {
		name   string
		labels []event.Label
	}{
		{
			name: "bytes label",
			labels: []event.Label{
				event.Bytes("data", []byte("hello world")),
			},
		},
		{
			name: "bool labels",
			labels: []event.Label{
				event.Bool("enabled", true),
				event.Bool("disabled", false),
			},
		},
		{
			name: "interface string",
			labels: []event.Label{
				event.Value("str_interface", "hello"),
			},
		},
		{
			name: "interface stringer",
			labels: []event.Label{
				event.Value("stringer", time.Duration(5*time.Second)),
			},
		},
		{
			name: "interface other",
			labels: []event.Label{
				event.Value("complex", complex(1, 2)),
				event.Value("slice", []int{1, 2, 3}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			// Create event manually to test label function paths
			ev := &event.Event{
				At:     time.Now(),
				Kind:   event.LogKind,
				Labels: append([]event.Label{event.String("msg", "test message")}, tt.labels...),
			}

			handler.Event(ctx, ev)

			// Should produce valid output without panic
			output := buf.String()
			assert.Contains(t, output, "test message")
		})
	}
}
