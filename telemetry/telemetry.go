package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"

	"golang.org/x/exp/event"
)

var (
	ErrNilHandler = errors.New("handler cannot be nil")
)

type handler interface {
	Event(ctx context.Context, e *event.Event) context.Context
}

// MultiHandler aggregates multiple event handlers and forwards events to all of them.
// It implements the event.Handler interface.
type MultiHandler struct {
	Metric handler
	Trace  handler
	Log    handler
}

var _ event.Handler = (*MultiHandler)(nil)

// Event implements the event.Handler interface.
// It forwards the event to all non-nil handlers in the order: Log, Metric, Trace.
func (h *MultiHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	if ev == nil {
		return ctx
	}

	// Process in order: Log first (for debugging), then metrics, then traces
	if h.Log != nil {
		ctx = h.Log.Event(ctx, ev)
	}

	if h.Metric != nil {
		ctx = h.Metric.Event(ctx, ev)
	}

	if h.Trace != nil {
		ctx = h.Trace.Event(ctx, ev)
	}

	return ctx
}

// Close closes all handlers that implement io.Closer.
// It returns the first error encountered, but continues closing all handlers.
func (h *MultiHandler) Close() error {
	var firstErr error

	if closer, ok := h.Log.(io.Closer); ok && closer != nil {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close log handler: %w", err)
		}
	}

	if closer, ok := h.Metric.(io.Closer); ok && closer != nil {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close metric handler: %w", err)
		}
	}

	if closer, ok := h.Trace.(io.Closer); ok && closer != nil {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close trace handler: %w", err)
		}
	}

	return firstErr
}
