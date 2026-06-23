package tracker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
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

func NewWithLogger(ctx context.Context, logger *slog.Logger, attrs ...slog.Attr) *Tracker {
	t := &Tracker{
		ctx:    ctx,
		logger: logger,
		start:  time.Now(),
	}
	t.SetAttrs(attrs...)
	return t
}

// New creates and returns a new Tracker instance.
// It initializes the timer and sets the initial message context.
//
// ctx: The context to be used for all logging operations within this tracker.
// msg: The primary message to be logged when the operation completes successfully.
// attrs: Any initial attributes to be associated with this operation.
func New(ctx context.Context, attrs ...slog.Attr) *Tracker {
	t := &Tracker{
		ctx:    ctx,
		logger: slog.Default(),
		start:  time.Now(),
	}
	t.SetAttrs(attrs...)
	return t
}

func (t *Tracker) Debug(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelDebug, msg, attrs...)
}

func (t *Tracker) Error(err error, attrs ...slog.Attr) error {
	t.once.Do(func() {
	})
	t.Log(t.ctx, 3, slog.LevelError, err.Error(), slog.Duration("took", time.Since(t.start)))
	return err
}

func (t *Tracker) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)

	t.once.Do(func() {
	})
	t.Log(t.ctx, 3, slog.LevelError, err.Error(), slog.Duration("took", time.Since(t.start)))
	return err
}

func (t *Tracker) Info(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelInfo, msg, attrs...)
}

func (t *Tracker) Warn(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelWarn, msg, attrs...)
}

func (t *Tracker) Log(ctx context.Context, depth int, level slog.Level, msg string, attrs ...slog.Attr) {
	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])
	pc = pcs[0]

	// Create a new Record
	// Signature: NewRecord(time time.Time, level Level, msg string, pc uintptr)
	record := slog.NewRecord(time.Now(), level, msg, pc)
	record.AddAttrs(attrs...)
	_ = t.logger.Handler().Handle(ctx, record)
}

// SetAttrs appends a variable list of attributes to the tracker's context.
// Use this to attach metadata that should be part of the final Log record.
// We do not use "With" to avoid passing sync.Once.
func (t *Tracker) SetAttrs(attrs ...slog.Attr) {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	t.logger = t.logger.With(args...)
}

// Done logs the message with duration.
func (t *Tracker) Done(msg string, attrs ...slog.Attr) {
	t.once.Do(func() {
		t.Log(t.ctx, 6, slog.LevelInfo, msg, append(attrs, slog.Duration("took", time.Since(t.start)))...)
	})
}
