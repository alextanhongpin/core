// Package retry implements retry mechanism with throttler.
package retry

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrThrottled     = errors.New("retry: throttled")
)

type Handler struct {
	Options *Options
}

func New(opts ...Option) *Handler {
	return &Handler{
		Options: NewOptions().Apply(opts...),
	}
}

func (h *Handler) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	seq, done := try(ctx, h.Options)
	for range seq {
		err = fn(ctx)
		if err == nil {
			return
		}
	}

	return errors.Join(err, done())
}

func Try(ctx context.Context, opts ...Option) (iter.Seq[int], func() error) {
	return try(ctx, NewOptions().Apply(opts...))
}

func try(ctx context.Context, opt *Options) (iter.Seq[int], func() error) {
	var iterErr error
	return func(yield func(int) bool) {
			for i := range opt.Attempts + 1 {
				opt.MetricsCollector.IncAttempts()
				// Throttle only applies to retries, skip the first call.
				if i != 0 && !opt.Throttler.Allow() {
					opt.MetricsCollector.IncThrottles()
					iterErr = ErrThrottled
					break
				}

				if !yield(i) {
					opt.MetricsCollector.IncSuccesses()
					// Breaking early is considered a success.
					opt.Throttler.Success()
					break
				}

				// Can be disabled by setting attempts = 0.
				// Should not be treated as error.
				if i == opt.Attempts && opt.Attempts > 0 {
					opt.MetricsCollector.IncLimitExceeded()
					iterErr = ErrLimitExceeded
					break
				}

				opt.MetricsCollector.IncFailures()
				// Using time.Sleep blocks the operation and cannot be cancelled in case
				// timeout becomes very long.
				// Use time.After combined with context instead.
				select {
				case <-ctx.Done():
					iterErr = context.Cause(ctx)
					// Cannot break in select, return instead.
					return
				case <-time.After(opt.Backoff.At(i)):
				}
			}
		}, func() error {
			return iterErr
		}
}

func Do(ctx context.Context, fn func(context.Context) error, opts ...Option) (err error) {
	seq, done := Try(ctx, opts...)
	for i := range seq {
		err = fn(Attempts.WithValue(ctx, i))
		if err == nil {
			return nil
		}
	}

	return errors.Join(err, done())
}

func DoValue[T any](ctx context.Context, fn func(context.Context) (T, error), opts ...Option) (v T, err error) {
	seq, done := Try(ctx, opts...)
	for i := range seq {
		v, err = fn(Attempts.WithValue(ctx, i))
		if err == nil {
			return
		}
	}

	return v, errors.Join(err, done())
}

var Attempts contextKey[int] = "attempts"

type contextKey[T any] string

func (key contextKey[T]) WithValue(ctx context.Context, v T) context.Context {
	return context.WithValue(ctx, key, v)
}

func (key contextKey[T]) Value(ctx context.Context) (T, bool) {
	v, ok := ctx.Value(key).(T)
	return v, ok
}

func (key contextKey[T]) MustValue(ctx context.Context) T {
	v, ok := ctx.Value(key).(T)
	if !ok {
		panic(fmt.Errorf("contextKey: not present: %q", key))
	}
	return v
}
