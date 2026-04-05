package tracker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Tracker struct {
	ctx   context.Context
	attrs []slog.Attr
	err   error
	msg   string
	start time.Time
}

func New(ctx context.Context, msg string, attrs ...slog.Attr) *Tracker {
	return &Tracker{
		ctx:   ctx,
		attrs: attrs,
		msg:   msg,
		start: time.Now(),
	}
}

func (t *Tracker) Error(err error) error {
	t.err = err
	return t.err
}

func (t *Tracker) Errorf(msg string, args ...any) error {
	t.err = fmt.Errorf(msg, args...)
	return t.err
}

func (t *Tracker) Attrs(attrs ...slog.Attr) {
	t.attrs = append(t.attrs, attrs...)
}

func (t *Tracker) Infof(msg string, args ...any) {
	slog.LogAttrs(t.ctx, slog.LevelInfo, fmt.Sprintf(msg, args...), t.attrs...)
}

func (t *Tracker) Info(msg string, attrs ...slog.Attr) {
	attrs = append(t.attrs, attrs...)
	slog.LogAttrs(t.ctx, slog.LevelInfo, msg, attrs...)
}

func (t *Tracker) Done() {
	attrs := append(t.attrs, slog.Duration("duration", time.Since(t.start)))

	if err := t.err; err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		slog.LogAttrs(t.ctx, slog.LevelError, t.msg, attrs...)
	} else {
		slog.LogAttrs(t.ctx, slog.LevelInfo, t.msg, attrs...)
	}
}
