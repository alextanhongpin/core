package tracker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type tracker interface {
	Debug(msg string, attrs ...slog.Attr)
	Done(msg string, attrs ...slog.Attr)
	Error(err error, attrs ...slog.Attr) error
	Errorf(format string, args ...any) error
	Info(msg string, attrs ...slog.Attr)
	Log(ctx context.Context, depth int, level slog.Level, msg string, attrs ...slog.Attr)
	SetAttrs(attrs ...slog.Attr)
	Warn(msg string, attrs ...slog.Attr)
}

var _ tracker = (*Tracker)(nil)

// Tracker logs the steps and outcome of one timed operation. Every record
// carries the attributes given to New and SetAttrs, and the source location
// of the Tracker method's caller.
//
// The outcome is logged once: the first Done logs msg with a "took"
// duration, and a call to Error or Errorf before it suppresses Done. Logging
// methods are safe for concurrent use; SetAttrs is not.
type Tracker struct {
	ctx    context.Context
	logger *slog.Logger
	once   sync.Once
	start  time.Time
}

// NewWithLogger returns a Tracker that logs to logger with ctx, starting the
// timer now.
func NewWithLogger(ctx context.Context, logger *slog.Logger, attrs ...slog.Attr) *Tracker {
	t := &Tracker{
		ctx:    ctx,
		logger: logger,
		start:  time.Now(),
	}
	t.SetAttrs(attrs...)
	return t
}

// New is NewWithLogger with slog.Default().
func New(ctx context.Context, attrs ...slog.Attr) *Tracker {
	return NewWithLogger(ctx, slog.Default(), attrs...)
}

// Debug logs msg at debug level.
func (t *Tracker) Debug(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelDebug, msg, attrs...)
}

// Info logs msg at info level.
func (t *Tracker) Info(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelInfo, msg, attrs...)
}

// Warn logs msg at warn level.
func (t *Tracker) Warn(msg string, attrs ...slog.Attr) {
	t.Log(t.ctx, 3, slog.LevelWarn, msg, attrs...)
}

// Done logs msg at info level with the elapsed time as "took". Only the
// first call logs, and nothing is logged if Error or Errorf ran first.
func (t *Tracker) Done(msg string, attrs ...slog.Attr) {
	pc := callerPC(2)
	t.once.Do(func() {
		t.log(pc, slog.LevelInfo, msg, append(attrs, t.took())...)
	})
}

// Error logs err at error level with attrs and "took", suppresses a later
// Done, and returns err. Every call logs. A nil err logs nothing and returns
// nil.
func (t *Tracker) Error(err error, attrs ...slog.Attr) error {
	if err == nil {
		return nil
	}
	t.once.Do(func() {})
	t.log(callerPC(2), slog.LevelError, err.Error(), append(attrs, t.took())...)
	return err
}

// Errorf is Error(fmt.Errorf(format, args...)).
func (t *Tracker) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	t.once.Do(func() {})
	t.log(callerPC(2), slog.LevelError, err.Error(), t.took())
	return err
}

// Log writes a record at level using ctx. depth is passed to
// runtime.Callers to find the source location: 2 is the caller of Log.
func (t *Tracker) Log(ctx context.Context, depth int, level slog.Level, msg string, attrs ...slog.Attr) {
	if !t.logger.Enabled(ctx, level) {
		return
	}
	t.logCtx(ctx, callerPC(depth), level, msg, attrs...)
}

// SetAttrs adds attributes to every later record from t. It must not be
// called concurrently with other methods.
func (t *Tracker) SetAttrs(attrs ...slog.Attr) {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	t.logger = t.logger.With(args...)
}

func (t *Tracker) took() slog.Attr {
	return slog.Duration("took", time.Since(t.start))
}

func (t *Tracker) log(pc uintptr, level slog.Level, msg string, attrs ...slog.Attr) {
	if !t.logger.Enabled(t.ctx, level) {
		return
	}
	t.logCtx(t.ctx, pc, level, msg, attrs...)
}

func (t *Tracker) logCtx(ctx context.Context, pc uintptr, level slog.Level, msg string, attrs ...slog.Attr) {
	record := slog.NewRecord(time.Now(), level, msg, pc)
	record.AddAttrs(attrs...)
	_ = t.logger.Handler().Handle(ctx, record)
}

// callerPC returns the pc skip frames above callerPC: 1 is its caller.
func callerPC(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(skip+1, pcs[:])
	return pcs[0]
}
