package tracker_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alextanhongpin/core/types/tracker"
)

var bb = new(bytes.Buffer)

func init() {
	// 1. Initialize with a base handler (e.g., Text)
	baseHandler := slog.NewTextHandler(bb, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "took" {
				a.Value = slog.DurationValue(0)
			}
			if a.Key == slog.TimeKey {
				a.Value = slog.TimeValue(time.Time{})
			}
			return a
		},
	})
	logger := slog.New(&ReqIDHandler{Handler: baseHandler})
	slog.SetDefault(logger)
}

func ExampleNew_error() {
	bb.Reset()
	// 2. Add Request ID to context (usually done in HTTP middleware)
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123abc")

	_ = foo(ctx, errors.ErrUnsupported)
	fmt.Println(bb.String())

	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=foo key=value request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=retry call=api request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=0 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=1 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=2 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retry call=api took=0s request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=ERROR msg=foo key=value took=0s cause="unsupported operation" request_id=req-123abc
}

func ExampleNew_info() {
	bb.Reset()

	// 2. Add Request ID to context (usually done in HTTP middleware)
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123abc")

	_ = foo(ctx, nil)
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=foo key=value request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=retry call=api request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=0 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=1 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... call=api attempts=2 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retry call=api took=0s request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO msg=foo key=value took=0s request_id=req-123abc
}

func foo(ctx context.Context, err error) error {
	t := tracker.New(ctx, "foo", slog.String("key", "value"))
	defer t.Done()

	tt := tracker.New(ctx, "retry", slog.String("call", "api"))
	for i := range 3 {
		tt.Info("retrying...", slog.Int("attempts", i))
	}
	tt.Done()

	if err != nil {
		return t.Error(err)
	}

	return nil
}

type contextKey string

const requestIDKey contextKey = "request_id"

// ReqIDHandler wraps an existing handler to inject context values
type ReqIDHandler struct {
	slog.Handler
}

func (h *ReqIDHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract ID from context
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}
