// Package retry implements retry mechanism with throttler.
package retry

import (
	"cmp"
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

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type retry interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// Func wraps a function with retry, backoff, and throttling capabilities.
func Func[K, V any](fn fun[K, V], rt retry) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		err = rt.Do(ctx, func(ctx context.Context) error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}

type Retry struct {
	*Config
}

func New(cfg *Config) *Retry {
	return &Retry{
		Config: cmp.Or(cfg, DefaultConfig()),
	}
}

func (r *Retry) Do(ctx context.Context, fn func(context.Context) error) error {
	retryable := r.Retryable
	attempts := r.Attempts
	backoff := r.Backoff
	throttler := r.Throttler

	var errs []error
	for i := range attempts + 1 {
		if i != 0 {
			if !throttler.Allow() {
				return errors.Join(append(errs, ErrThrottled)...)
			}

			d := backoff.At(i)
			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				timer.Stop()
				return context.Cause(ctx)
			case <-timer.C:
			}
			// timer is stopped by GC after firing; stop explicitly to avoid leak
			timer.Stop()
		}

		err := fn(ctx)
		if err == nil {
			throttler.Success()
			return nil
		}
		if cause, ok := retryable(err); !ok {
			return cause
		}
		errs = append(errs, err)
	}

	return errors.Join(append(errs, fmt.Errorf("%w: retried %d times", ErrLimitExceeded, attempts))...)
}
