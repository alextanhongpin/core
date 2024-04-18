// Package retry implements functions for retry mechanism.
package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

var ErrTooManyAttempts = errors.New("retry: too many attempts")

type Event struct {
	StartAt time.Time
	RetryAt time.Time
	Attempt int
	Delay   time.Duration
	Err     error
}

type Result struct {
	Retries []time.Time
}

type Retry struct {
	backoffs   []time.Duration
	OnRetry    func(Event)
	JitterFunc func(time.Duration) time.Duration
	Now        func() time.Time
}

func New(backoffs ...time.Duration) *Retry {
	if len(backoffs) == 0 {
		backoffs = []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			400 * time.Millisecond,
			800 * time.Millisecond,
			1600 * time.Millisecond,
			3200 * time.Millisecond,
			6400 * time.Millisecond,
			12800 * time.Millisecond,
			25600 * time.Millisecond,
			51200 * time.Millisecond,
		}
	}

	// The first execution does not count as retry.
	backoffs = append([]time.Duration{0}, backoffs...)

	return &Retry{
		backoffs: backoffs,
		OnRetry:  func(Event) {},
		JitterFunc: func(d time.Duration) time.Duration {
			n := int(d)
			if n == 0 {
				return 0
			}

			return time.Duration(n/2 + rand.Intn(n/2))
		},
		Now: time.Now,
	}
}

func (r *Retry) Do(ctx context.Context, fn func(ctx context.Context) error) (res *Result, err error) {
	start := r.Now()
	res = new(Result)

	for i, t := range r.backoffs {
		if i != 0 {
			select {
			case <-ctx.Done():
				return res, context.Cause(ctx)
			default:
				t = r.JitterFunc(t)
				time.Sleep(t)
			}

			res.Retries = append(res.Retries, r.Now())

			// Useful for recording metrics.
			r.OnRetry(Event{
				StartAt: start,
				RetryAt: res.Retries[len(res.Retries)-1],
				Attempt: i,
				Delay:   t,
				Err:     err,
			})
		}

		err = fn(ctx)
		if err == nil {
			return
		}
	}

	return res, errors.Join(ErrTooManyAttempts, err)
}
