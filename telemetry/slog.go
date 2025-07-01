package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/event"
	"golang.org/x/exp/slog"
)

var (
	ErrNilLogger = errors.New("logger cannot be nil")
)

type SlogHandler struct {
	logger       *slog.Logger
	errorHandler func(error)
}

// SlogHandlerOption configures an SlogHandler.
type SlogHandlerOption func(*SlogHandler)

// WithSlogErrorHandler sets a custom error handler for the SlogHandler.
// If not set, errors will be logged using the default logger.
func WithSlogErrorHandler(handler func(error)) SlogHandlerOption {
	return func(s *SlogHandler) {
		s.errorHandler = handler
	}
}

func NewSlogHandler(logger *slog.Logger, opts ...SlogHandlerOption) (*SlogHandler, error) {
	if logger == nil {
		return nil, ErrNilLogger
	}

	handler := &SlogHandler{
		logger: logger,
		errorHandler: func(err error) {
			log.Printf("SlogHandler error: %v", err)
		},
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

func (h *SlogHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	if ev == nil {
		h.handleError(ErrNilEvent)
		return ctx
	}

	if ev.Kind != event.LogKind {
		return ctx
	}

	var attrs []slog.Attr

	if ev.Source.Space != "" {
		attrs = append(attrs, slog.String("in", ev.Source.Space))
	}

	if ev.Source.Owner != "" {
		attrs = append(attrs, slog.String("owner", ev.Source.Owner))
	}

	if ev.Source.Name != "" {
		attrs = append(attrs, slog.String("name", ev.Source.Name))
	}
	if ev.Parent != 0 {
		attrs = append(attrs, slog.Uint64("parent", ev.Parent))
	}

	var isError bool
	var msg string

	// Find message first
	msgLabel := ev.Find("msg")
	if msgLabel.HasValue() {
		msg = msgLabel.String()
	}

	for _, l := range ev.Labels {
		if !l.HasValue() || l.Name == "" || l.Name == "msg" {
			continue
		}

		if l.Name == "error" {
			isError = true
		}

		attr := label(l)
		if attr.Key != "" { // Only add valid attributes
			attrs = append(attrs, attr)
		}
	}

	level := slog.LevelInfo
	if isError {
		level = slog.LevelError
	}

	// https://github.com/uptrace/opentelemetry-go-extra/blob/main/otellogrus/otellogrus.go#L91
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		// Adds TraceIds and SpanIds to logs.
		spanCtx := span.SpanContext()
		if spanCtx.HasTraceID() {
			attrs = append(attrs, slog.String("traceId", spanCtx.TraceID().String()))
		}

		if spanCtx.HasSpanID() {
			attrs = append(attrs, slog.String("spanId", spanCtx.SpanID().String()))
		}

		if isError {
			span.SetStatus(codes.Error, msg)
		}
	}

	h.logAttrs(ctx, ev.At, level, msg, attrs...)

	return ctx
}

func (h *SlogHandler) handleError(err error) {
	if h.errorHandler != nil {
		h.errorHandler(err)
	}
}

func (h *SlogHandler) logAttrs(ctx context.Context, at time.Time, level slog.Level, msg string, attrs ...slog.Attr) {
	l := h.logger
	if !l.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	// skip [runtime.Callers, this function logAttrs, this function's caller Event]
	runtime.Callers(7, pcs[:])
	pc := pcs[0]
	r := slog.NewRecord(at, level, msg, pc)
	r.AddAttrs(attrs...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.Handler().Handle(ctx, r)
}

func label(l event.Label) slog.Attr {
	if !l.HasValue() || l.Name == "" {
		return slog.Attr{}
	}

	switch {
	case l.IsString():
		return slog.String(l.Name, l.String())
	case l.IsBytes():
		return slog.String(l.Name, string(l.Bytes()))
	case l.IsInt64():
		return slog.Int64(l.Name, l.Int64())
	case l.IsUint64():
		return slog.Uint64(l.Name, l.Uint64())
	case l.IsFloat64():
		return slog.Float64(l.Name, l.Float64())
	case l.IsBool():
		return slog.Bool(l.Name, l.Bool())
	default:
		v := l.Interface()
		switch v := v.(type) {
		case string:
			return slog.String(l.Name, v)
		case fmt.Stringer:
			return slog.String(l.Name, v.String())
		default:
			return slog.Any(l.Name, v)
		}
	}
}
