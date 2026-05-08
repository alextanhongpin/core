package tracker

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Tracker is a utility struct designed to wrap the execution of a timed operation.
// It collects contextual attributes, tracks an error if one occurs, and logs
// a final summary message upon calling Done().
type Tracker struct {
	ctx    context.Context
	logger *slog.Logger
	once   sync.Once
	start  time.Time
}

// New creates and returns a new Tracker instance.
// It initializes the timer and sets the initial message context.
//
// ctx: The context to be used for all logging operations within this tracker.
// msg: The primary message to be logged when the operation completes successfully.
// attrs: Any initial attributes to be associated with this operation.
func New(ctx context.Context, logger *slog.Logger, attrs ...slog.Attr) *Tracker {
	t := &Tracker{
		ctx:    ctx,
		logger: cmp.Or(logger, slog.Default()),
		start:  time.Now(),
	}
	return t.WithAttrs(attrs...)
}

func (t *Tracker) Debug(msg string, attrs ...slog.Attr) *Tracker {
	t.logger.LogAttrs(t.ctx, slog.LevelDebug, msg, attrs...)
	return t
}

func (t *Tracker) Error(err error, attrs ...slog.Attr) error {
	t.once.Do(func() {
	})
	t.logger.LogAttrs(t.ctx, slog.LevelError, err.Error(), append(attrs, slog.Duration("took", time.Since(t.start)))...)
	return err
}

func (t *Tracker) Errorf(format string, args ...any) error {
	return t.Error(fmt.Errorf(format, args...))
}

func (t *Tracker) Info(msg string, attrs ...slog.Attr) *Tracker {
	t.logger.LogAttrs(t.ctx, slog.LevelInfo, msg, attrs...)
	return t
}

func (t *Tracker) Warn(msg string, attrs ...slog.Attr) *Tracker {
	t.logger.LogAttrs(t.ctx, slog.LevelWarn, msg, attrs...)
	return t
}

// WithAttrs appends a variable list of attributes to the tracker's context.
// Use this to attach metadata that should be part of the final log record.
// It returns the tracker pointer for chaining.
func (t *Tracker) WithAttrs(attrs ...slog.Attr) *Tracker {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	t.logger = t.logger.With(args...)
	return t
}

// Done logs the message with duration.
func (t *Tracker) Done(msg string, attrs ...slog.Attr) {
	t.once.Do(func() {
		t.logger.LogAttrs(t.ctx, slog.LevelInfo, msg, append(attrs, slog.Duration("took", time.Since(t.start)))...)
	})
}
