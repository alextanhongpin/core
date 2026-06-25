package tracker

import (
	"context"
	"log/slog"
)

var _ tracker = (*Noop)(nil)

type Noop struct {
}

func NewNoop() *Noop {
	return new(Noop)
}

func (n *Noop) Debug(msg string, attrs ...slog.Attr) {
}
func (n *Noop) Done(msg string, attrs ...slog.Attr) {}
func (n *Noop) Error(err error, attrs ...slog.Attr) error {
	return nil
}
func (n *Noop) Errorf(format string, args ...any) error {
	return nil
}
func (n *Noop) Info(msg string, attrs ...slog.Attr) {}
func (n *Noop) Log(ctx context.Context, depth int, level slog.Level, msg string, attrs ...slog.Attr) {
}
func (n *Noop) SetAttrs(attrs ...slog.Attr)         {}
func (n *Noop) Warn(msg string, attrs ...slog.Attr) {}
