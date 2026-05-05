package tracker_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alextanhongpin/core/types/tracker"
)

var bb = new(bytes.Buffer)

func init() {
	// 1. Initialize with a base handler (e.g., Text)
	next := slog.NewTextHandler(bb, &slog.HandlerOptions{
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
	logger := slog.New(&ReqIDHandler{Next: next})
	slog.SetDefault(logger)
}

func ExampleNew_error() {
	bb.Reset()

	_ = foo(context.Background(), errors.ErrUnsupported, slog.Default())
	fmt.Println(bb.String())

	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=foo
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=retry
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=0 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=1 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=2 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=ERROR msg=fail took=0s detail="unsupported operation" op=foo key=value
}

func ExampleNew_success() {
	bb.Reset()

	_ = foo(context.Background(), nil, nil)
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=foo
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=retry
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=0 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=1 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=2 op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s op=foo key=value
}

func ExampleNew_requestID() {
	bb.Reset()

	// 2. Add Request ID to context (usually done in HTTP middleware)
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123abc")

	_ = foo(ctx, nil, nil)
	fmt.Println(bb.String())

	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init request_id=req-123abc op=foo
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init request_id=req-123abc op=retry
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=0 request_id=req-123abc op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=1 request_id=req-123abc op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... attempts=2 request_id=req-123abc op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s request_id=req-123abc op=retry api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s request_id=req-123abc op=foo key=value
}

func ExampleNew_discard() {
	bb.Reset()

	_ = foo(context.Background(), nil, slog.New(slog.DiscardHandler))
	fmt.Println(bb.String())
	// Output:
	//
}

func ExampleNew_group() {
	bb.Reset()

	_ = foo(context.Background(), nil, slog.Default().WithGroup("my_group"))
	fmt.Println(bb.String())

	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init my_group.op=foo
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init my_group.op=retry
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... my_group.attempts=0 my_group.op=retry my_group.api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... my_group.attempts=1 my_group.op=retry my_group.api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=retrying... my_group.attempts=2 my_group.op=retry my_group.api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done my_group.took=0s my_group.op=retry my_group.api=bar
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done my_group.took=0s my_group.op=foo my_group.key=value
}

func ExampleNew_idempotent() {
	bb.Reset()

	ctx := context.Background()
	t := tracker.New(ctx, nil, "foo").WithAttrs(slog.String("key", "value"))
	t.Done()
	t.Done()
	t.Done()
	_ = t.Errorf("bad request")

	tt := tracker.New(ctx, nil, "foo").WithAttrs(slog.String("key", "value"))
	_ = tt.Errorf("error 1")
	_ = tt.Errorf("error 2")
	_ = tt.Errorf("error 3")
	tt.Done()
	fmt.Println(bb.String())

	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=foo
	// time=0001-01-01T00:00:00.000Z level=INFO msg=done took=0s op=foo key=value
	// time=0001-01-01T00:00:00.000Z level=DEBUG msg=init op=foo
	// time=0001-01-01T00:00:00.000Z level=ERROR msg=fail took=0s detail="error 1" op=foo key=value
}

func foo(ctx context.Context, err error, logger *slog.Logger) error {
	t := tracker.New(ctx, logger, "foo").WithAttrs(slog.String("key", "value"))
	defer t.Done()

	tt := tracker.New(ctx, logger, "retry").WithAttrs(slog.String("api", "bar"))
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
	Next  slog.Handler
	group string
	attrs []slog.Attr
}

func (h *ReqIDHandler) Handle(ctx context.Context, r slog.Record) error {
	rc := r.Clone()
	// Extract ID from context
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		rc.AddAttrs(slog.String("request_id", id))
	}
	rc.AddAttrs(h.attrs...)

	next := h.Next
	if h.group != "" {
		next = next.WithGroup(h.group)
	}
	return next.Handle(ctx, rc)
}

func (h *ReqIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// NOTE: Returning h.Next.WithAttrs(attrs) will result in h.Handle using the
	// original h.Next.Handle.
	c := h.clone()
	c.attrs = append(h.attrs, attrs...)
	return c
}

func (h *ReqIDHandler) WithGroup(name string) slog.Handler {
	c := h.clone()
	c.group = name
	return c
}

func (h *ReqIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Next.Enabled(ctx, level)
}

func (h *ReqIDHandler) clone() *ReqIDHandler {
	return &ReqIDHandler{
		Next:  h.Next,
		group: h.group,
		attrs: slices.Clone(h.attrs),
	}
}

func Op(v string) slog.Attr {
	return slog.String("op", v)
}
