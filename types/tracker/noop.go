package tracker

import (
	"context"
	"fmt"
	"log/slog"
)

var _ tracker = (*Noop)(nil)

// Noop has the methods of Tracker and logs nothing. Error and Errorf still
// return the error, so `return t.Error(err)` behaves the same with either.
type Noop struct{}

// NewNoop returns a Noop.
func NewNoop() *Noop {
	return new(Noop)
}

func (*Noop) Debug(string, ...slog.Attr)                                 {}
func (*Noop) Done(string, ...slog.Attr)                                  {}
func (*Noop) Info(string, ...slog.Attr)                                  {}
func (*Noop) Warn(string, ...slog.Attr)                                  {}
func (*Noop) SetAttrs(...slog.Attr)                                      {}
func (*Noop) Log(context.Context, int, slog.Level, string, ...slog.Attr) {}
func (*Noop) Error(err error, _ ...slog.Attr) error                      { return err }
func (*Noop) Errorf(format string, args ...any) error                    { return fmt.Errorf(format, args...) }
