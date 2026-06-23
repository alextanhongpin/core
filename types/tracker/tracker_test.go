package tracker_test

import (
	"bytes"
	"cmp"
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
		AddSource: true,
		Level:     slog.LevelDebug,
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
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:138 msg=foo key=value
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=0
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=1
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=2
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:145 msg=done api=bar took=0s
	// time=0001-01-01T00:00:00.000Z level=ERROR source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:148 msg="unsupported operation" key=value took=0s
}

func ExampleNew_success() {
	bb.Reset()

	_ = foo(context.Background(), nil, nil)
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:138 msg=foo key=value
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=0
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=1
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=2
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:145 msg=done api=bar took=0s
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:151 msg=done key=value took=0s
}

func ExampleNew_requestID() {
	bb.Reset()

	// 2. Add Request ID to context (usually done in HTTP middleware)
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123abc")

	_ = foo(ctx, nil, nil)
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:138 msg=foo key=value request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=0 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=1 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... api=bar attempts=2 request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:145 msg=done api=bar took=0s request_id=req-123abc
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:151 msg=done key=value took=0s request_id=req-123abc
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

	_ = foo(context.Background(), nil, slog.Default().WithGroup("group"))
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:138 msg=foo group.key=value
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... group.api=bar group.attempts=0
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... group.api=bar group.attempts=1
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:143 msg=retrying... group.api=bar group.attempts=2
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:145 msg=done group.api=bar group.took=0s
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:151 msg=done group.key=value group.took=0s
}

func ExampleNew_idempotent() {
	bb.Reset()

	ctx := context.Background()
	t := tracker.New(ctx, slog.String("key", "value"))
	t.Debug("init")
	for range 3 {
		t.Done("done")
	}
	_ = t.Errorf("bad request")

	tt := tracker.New(ctx)
	tt.SetAttrs(slog.String("key", "value"))
	tt.Debug("retrying")
	for i := range 3 {
		_ = tt.Error(errors.New("retry error"), slog.Int("attempt", i+1))
	}
	tt.Done("retry ok")
	fmt.Println(bb.String())
	// Output:
	// time=0001-01-01T00:00:00.000Z level=DEBUG source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:111 msg=init key=value
	// time=0001-01-01T00:00:00.000Z level=INFO source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:113 msg=done key=value took=0s
	// time=0001-01-01T00:00:00.000Z level=ERROR source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:115 msg="bad request" key=value took=0s
	// time=0001-01-01T00:00:00.000Z level=DEBUG source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:119 msg=retrying key=value
	// time=0001-01-01T00:00:00.000Z level=ERROR source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:121 msg="retry error" key=value took=0s
	// time=0001-01-01T00:00:00.000Z level=ERROR source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:121 msg="retry error" key=value took=0s
	// time=0001-01-01T00:00:00.000Z level=ERROR source=/Users/alextanhongpin/Documents/go/core/types/tracker/tracker_test.go:121 msg="retry error" key=value took=0s
}

func foo(ctx context.Context, err error, logger *slog.Logger) error {
	logger = cmp.Or(logger, slog.Default())
	t := tracker.NewWithLogger(ctx, logger, slog.String("key", "value"))
	t.Info("foo")
	defer t.Done("done")

	tt := tracker.NewWithLogger(ctx, logger, slog.String("api", "bar"))
	for i := range 3 {
		tt.Info("retrying...", slog.Int("attempts", i))
	}
	tt.Done("done")

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
	// Extract ID from context
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	next := h.Next
	if h.group != "" {
		next = next.WithGroup(h.group)
	}
	return next.WithAttrs(h.attrs).Handle(ctx, r)
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
