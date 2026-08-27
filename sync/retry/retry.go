// Package retry implements retry mechanism with throttler.
package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrThrottled     = errors.New("retry: throttled")
	ErrCanceled      = errors.New("retry: canceled")
)

type handler[K, V any] = func(ctx context.Context, req K) (V, error)

// Handler wraps a function with retry, backoff, and throttling capabilities.
func Handler[K, V any](fn handler[K, V], opts ...Option) handler[K, V] {
	opt := NewOptions()
	for _, o := range opts {
		o(opt)
	}
	retryable := opt.Retryable
	attempts := opt.Attempts
	backoff := opt.Backoff
	throttler := opt.Throttler

	return func(ctx context.Context, req K) (V, error) {
		var zero V
		res, err := fn(ctx, req)
		if err == nil {
			throttler.Success()
			return res, nil
		}

		if cause, ok := retryable(err); !ok {
			return zero, cause
		}

		var errs []error
		for i := range attempts {
			if !throttler.Allow() {
				return zero, errors.Join(append(errs, ErrThrottled)...)
			}

			timer := time.NewTimer(backoff.At(i))
			select {
			case <-ctx.Done():
				timer.Stop()
				return zero, context.Cause(ctx)

			case <-timer.C:
				res, err = fn(ctx, req)
				if err == nil {
					throttler.Success()
					return res, nil
				}
				if cause, ok := retryable(err); !ok {
					return zero, cause
				}
				errs = append(errs, err)
			}
		}

		return zero, errors.Join(append(errs, fmt.Errorf("%w: retried %d times", ErrLimitExceeded, attempts))...)
	}
}

// Do executes fn with retry logic.
func Do(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	h := Handler(func(ctx context.Context, _ struct{}) (struct{}, error) {
		return struct{}{}, fn(ctx)
	}, opts...)
	_, err := h(ctx, struct{}{})
	return err
}

// DoValue executes fn with retry logic and returns the resulting value.
func DoValue[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	h := Handler(func(ctx context.Context, _ struct{}) (T, error) {
		return fn(ctx)
	}, opts...)
	return h(ctx, struct{}{})
}
