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
)

type handler[K, V any] = func(ctx context.Context, req K) (V, error)

func Handler[K, V any](fn handler[K, V], opts ...Option) handler[K, V] {
	return func(ctx context.Context, req K) (V, error) {
		var zero V
		res, err := fn(ctx, req)
		if err == nil {
			return res, nil
		}

		opt := OptionsFrom(opts...)
		attempts := opt.Attempts
		backoff := opt.Backoff
		throttler := opt.Throttler

		var errs []error
		for i := range attempts {
			if !throttler.Allow() {
				return zero, errors.Join(ErrThrottled, err)
			}

			select {
			case <-ctx.Done():
				return zero, context.Cause(ctx)

			case <-time.After(backoff.At(i)):
				res, err = fn(ctx, req)
				if err == nil {
					throttler.Success()
					return res, nil
				}
				errs = append(errs, err)
			}
		}

		return zero, errors.Join(append(errs, fmt.Errorf("%w: retried %d times", ErrLimitExceeded, attempts))...)
	}
}
