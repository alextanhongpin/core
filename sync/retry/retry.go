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

func Do[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	var zero T
	v, err := fn(ctx)
	if err == nil {
		return v, nil
	}

	opt := OptionsFrom(opts...)
	attempts := opt.Attempts
	backoff := opt.Backoff
	throttler := opt.Throttler

	for i := range attempts {
		if !throttler.Allow() {
			return zero, errors.Join(ErrThrottled, err)
		}
		duration := backoff.At(i)
		select {
		case <-time.After(duration):
			v, err = fn(ctx)
			if err == nil {
				throttler.Success()
				return v, nil
			}
			err = fmt.Errorf("retried %d times: %w", i+1, err)
		case <-ctx.Done():
			return zero, errors.Join(context.Cause(ctx), err)
		}
	}

	return zero, errors.Join(ErrLimitExceeded, err)
}

func Exec(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	_, err := Do(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	}, opts...)
	return err
}
