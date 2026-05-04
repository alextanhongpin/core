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
	// start records the time the tracking process began.
	*slog.Logger
	once  sync.Once
	start time.Time
}

// New creates and returns a new Tracker instance.
// It initializes the timer and sets the initial message context.
//
// ctx: The context to be used for all logging operations within this tracker.
// msg: The primary message to be logged when the operation completes successfully.
// attrs: Any initial attributes to be associated with this operation.
func New(logger *slog.Logger) *Tracker {
	return &Tracker{
		Logger: cmp.Or(logger, slog.Default()),
		start:  time.Now(),
	}
}

// Error sets the tracker's internal error state.
// If this method is called, the final log recorded by Done() will treat the operation as failed.
func (t *Tracker) Error(ctx context.Context, err error) error {
	t.done(ctx, "", err)
	return err
}

// Errorf sets the tracker's internal error state using formatted error messages.
// If this method is called, the final log recorded by Done() will treat the operation as failed.
func (t *Tracker) Errorf(ctx context.Context, format string, args ...any) error {
	return t.Error(ctx, fmt.Errorf(format, args...))
}

// With returns a Logger that includes the given attributes in each output
// operation. Arguments are converted to attributes as if by [Logger.Log].
func (t *Tracker) With(args ...any) *Tracker {
	t.Logger = t.Logger.With(args...)
	return t
}

// Attrs appends a variable list of attributes to the tracker's context.
// Use this to attach metadata that should be part of the final log record.
// It returns the tracker pointer for chaining.
func (t *Tracker) Attrs(attrs ...slog.Attr) *Tracker {
	a := make([]any, len(attrs))
	for i, attr := range attrs {
		a[i] = attr
	}

	return t.With(a...)
}

// Done logs the message with duration.
func (t *Tracker) Done(ctx context.Context, msg string) {
	t.done(ctx, msg, nil)
}

// Done finalizes the tracking process.
// It logs the final result (success or failure), including the total duration,
// and returns the tracker pointer for chaining.
// If err is non-nil, the log level will be ERROR.
// Otherwise, it logs at INFO level.
func (t *Tracker) done(ctx context.Context, msg string, err error) {
	t.once.Do(func() {
		// Calculate and append the total duration as the final attribute.
		tookAttr := slog.Duration("took", time.Since(t.start))

		if err != nil {
			// Log an error summary
			t.LogAttrs(ctx, slog.LevelError, msg, tookAttr, slog.String("cause", err.Error()))
		} else {
			// Log a success summary
			t.LogAttrs(ctx, slog.LevelInfo, msg, tookAttr)
		}
	})
}
