package telemetry

import (
	"context"

	"golang.org/x/exp/event"
)

type handler interface {
	Event(ctx context.Context, e *event.Event) context.Context
}

type MultiHandler struct {
	Metric handler
	Trace  handler
	Log    handler
}

func (h *MultiHandler) Event(ctx context.Context, ev *event.Event) context.Context {
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
