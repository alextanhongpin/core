package tracker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Tracker is a utility struct designed to wrap the execution of a timed operation.
// It collects contextual attributes, tracks an error if one occurs, and logs
// a final summary message upon calling Done().
type Tracker struct {
	ctx context.Context
	// attrs collects all attributes (key/value pairs) associated with this tracked operation.
	attrs []slog.Attr
	// err stores any error encountered during the tracked operation.
	err error
	// msg holds the primary message to be logged when Done() is called.
	msg string
	// start records the time the tracking process began.
	start time.Time
}

// New creates and returns a new Tracker instance.
// It initializes the timer and sets the initial message context.
//
// ctx: The context to be used for all logging operations within this tracker.
// msg: The primary message to be logged when the operation completes successfully.
// attrs: Any initial attributes to be associated with this operation.
func New(ctx context.Context, msg string, attrs ...slog.Attr) *Tracker {
	t := &Tracker{
		ctx:   ctx,
		attrs: attrs,
		msg:   msg,
		start: time.Now(),
	}
	return t.Debug(msg)
}

// Error sets the tracker's internal error state.
// If this method is called, the final log recorded by Done() will treat the operation as failed.
func (t *Tracker) Error(err error) error {
	t.err = err
	return t.err
}

// Errorf sets the tracker's internal error state using formatted error messages.
// If this method is called, the final log recorded by Done() will treat the operation as failed.
func (t *Tracker) Errorf(format string, args ...any) error {
	return t.Error(fmt.Errorf(format, args...))
}

// Attrs appends a variable list of attributes to the tracker's context.
// Use this to attach metadata that should be part of the final log record.
// It returns the tracker pointer for chaining.
func (t *Tracker) Attrs(attrs ...slog.Attr) *Tracker {
	t.attrs = append(t.attrs, attrs...)
	return t
}

// Info logs an informational message to the tracker's context.
// This is useful for logging interim steps *before* calling Done().
// It does not change the error state.
func (t *Tracker) Info(msg string, attrs ...slog.Attr) *Tracker {
	slog.LogAttrs(t.ctx, slog.LevelInfo, msg, append(t.attrs, attrs...)...)
	return t
}

// Infof logs an informational message using a format string.
// This is functionally similar to Info but kept for potential semantic difference
// if logging levels become more granular later.
// It does not change the error state.
func (t *Tracker) Infof(format string, args ...any) *Tracker {
	slog.LogAttrs(t.ctx, slog.LevelInfo, fmt.Sprintf(format, args...), t.attrs...)
	return t
}

// Debug logs a debug message to the tracker's context.
// This is useful for logging interim steps *before* calling Done().
// It does not change the error state.
func (t *Tracker) Debug(msg string, attrs ...slog.Attr) *Tracker {
	slog.LogAttrs(t.ctx, slog.LevelDebug, msg, append(t.attrs, attrs...)...)
	return t
}

// Debugf logs a debug message using a format string.
// This is functionally similar to Info but kept for potential semantic difference
// if logging levels become more granular later.
// It does not change the error state.
func (t *Tracker) Debugf(format string, args ...any) *Tracker {
	slog.LogAttrs(t.ctx, slog.LevelDebug, fmt.Sprintf(format, args...), t.attrs...)
	return t
}

// Done finalizes the tracking process.
// It logs the final result (success or failure), including the total duration,
// and returns the tracker pointer for chaining.
// If t.err is non-nil, the log level will be ERROR.
// Otherwise, it logs at INFO level.
func (t *Tracker) Done() {
	// Calculate and append the total duration as the final attribute.
	duration := time.Since(t.start)
	finalAttrs := append(t.attrs, slog.Duration("took", duration))

	if t.err != nil {
		// Log an error summary
		finalAttrs = append(finalAttrs, slog.String("cause", t.err.Error()))
		slog.LogAttrs(t.ctx, slog.LevelError, t.msg, finalAttrs...)
	} else {
		// Log a success summary
		slog.LogAttrs(t.ctx, slog.LevelInfo, t.msg, finalAttrs...)
	}

	// Resetting the error state might be desirable if the tracker is reused,
	// but for simplicity, we leave it as is, assuming Done() is the end of a cycle.
}
