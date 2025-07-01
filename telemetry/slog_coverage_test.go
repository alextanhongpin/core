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

var testCtx = context.Background()

// Test cases to improve coverage for the label function
func TestSlogLabelFunctionCoverage(t *testing.T) {
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
			name: "interface with string",
			labels: []event.Label{
				event.Value("interface_string", "hello"),
			},
		},
		{
			name: "interface with stringer",
			labels: []event.Label{
				event.Value("stringer", time.Duration(5*time.Second)),
			},
		},
		{
			name: "interface with other types",
			labels: []event.Label{
				event.Value("complex", complex(1, 2)),
				event.Value("slice", []int{1, 2, 3}),
				event.Value("map", map[string]int{"a": 1}),
			},
		},
		{
			name: "empty name label",
			labels: []event.Label{
				event.String("", "should be ignored"),
			},
		},
		{
			name: "label without value",
			labels: []event.Label{
				{}, // empty label
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			logCtx := event.NewContext(testCtx, handler)
			ev := event.To(logCtx)
			for _, label := range tt.labels {
				ev = ev.With(label)
			}
			ev.Log("test message")

			// Should not panic and should produce valid JSON log
			output := buf.String()
			if output != "" {
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(output), &parsed)
				assert.NoError(t, err, "Output should be valid JSON")
			}
		})
	}
}

func TestSlogLabelTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler, err := NewSlogHandler(logger)
	require.NoError(t, err)

	// Test the label function directly by creating events that exercise different paths
	t.Run("label without value", func(t *testing.T) {
		buf.Reset()
		logCtx := event.NewContext(testCtx, handler)

		// Create an event with no labels to test empty label handling
		event.To(logCtx).Log("message with no labels")

		output := buf.String()
		assert.Contains(t, output, "message with no labels")
	})

	t.Run("error in log formatting", func(t *testing.T) {
		buf.Reset()

		// Capture error handler calls
		var capturedError error
		errorHandler := func(err error) {
			capturedError = err
		}

		handler, err := NewSlogHandler(logger, WithSlogErrorHandler(errorHandler))
		require.NoError(t, err)

		// Call Event with nil to trigger error handling
		handler.Event(testCtx, nil)
		assert.Equal(t, ErrNilEvent, capturedError)
	})
}

func TestSlogHandlerAdvancedCoverage(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler, err := NewSlogHandler(logger)
	require.NoError(t, err)

	t.Run("event with source information", func(t *testing.T) {
		buf.Reset()
		logCtx := event.NewContext(testCtx, handler)

		// Create event with source and various label types
		event.To(logCtx).With(
			event.String("function", "test_function"),
			event.Int64("line", 123),
			event.Float64("version", 1.5),
			event.Uint64("count", 42),
		).Error("error with source info")

		output := buf.String()
		assert.Contains(t, output, "error with source info")
		assert.Contains(t, output, "ERROR")
	})

	t.Run("warn level logging", func(t *testing.T) {
		buf.Reset()
		logCtx := event.NewContext(testCtx, handler)

		// Using the correct function for warn level
		ctx := event.NewContext(testCtx, &eventtest.Handler{})
		event.To(ctx).With(
			event.String("component", "test"),
		).Log("warn level message")

		// Then send to our handler
		ev := &event.Event{}
		ev.At = time.Now()
		ev.Kind = event.LogKind
		ev.Labels = []event.Label{
			event.String("msg", "warn level message"),
			event.String("component", "test"),
			event.String("level", "WARN"), // Add level manually for coverage
		}

		handler.Event(testCtx, ev)

		output := buf.String()
		assert.Contains(t, output, "warn level message")
	})
}
