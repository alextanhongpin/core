// Package retry implements retry mechanism with throttler.
package retry

import (
	"context"
	"errors"
	"iter"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrThrottled     = errors.New("retry: throttled")
)

type retry interface {
	Try(ctx context.Context, limit int) iter.Seq2[int, error]
}

var _ retry = (*Retry)(nil)

type Retry struct {
	BackOffPolicy backOffPolicy
	Throttler     throttler
}

func New(bop backOffPolicy) *Retry {
	var t *Throttler

	return &Retry{
		BackOffPolicy: bop,
		Throttler:     t,
	}
}

func (r *Retry) Try(ctx context.Context, limit int) iter.Seq2[int, error] {
	return func(yield func(int, error) bool) {
		for i := range limit + 1 {
			if i == limit {
				yield(i, ErrLimitExceeded)

				break
			}

			// Throttle only applies to retries, skip the first call.
			if i > 0 && !r.Throttler.Allow() {
				yield(i, ErrThrottled)

				break
			}

			if err := ctx.Err(); err != nil {
				yield(i, err)

				return
			}

			if !yield(i, nil) {
				// Breaking early is considered a success.
				r.Throttler.Success()

				break
			}

			// Using time.Sleep blocks the operation and cannot be cancelled in case
			// timeout becomes very long.
			// Use time.After combined with context instead.
			select {
			case <-ctx.Done():
				break
			case <-time.After(r.BackOffPolicy.BackOff(i)):
			}
		}
	}
}
