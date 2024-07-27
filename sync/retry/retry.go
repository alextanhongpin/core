// Package retry implements functions for retry mechanism.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

var ErrRetryLimitExceeded = errors.New("retry: limit exceeded")
var ErrAborted = errors.New("retry: aborted")

type Retry struct {
	Attempts int
	Policy   func(ctx context.Context, attempts int) time.Duration
}

func New(attempts int) *Retry {
	if attempts <= 0 {
		panic("retry: attempts must be greater than 0")
	}

	return &Retry{
		Attempts: attempts,
		Policy:   ExponentialBackoff(100*time.Millisecond, 1*time.Minute, true),
	}
}

func (r *Retry) Do(fn func() error) (err error) {
	for i := range r.Attempts {
		if i != 0 {
			time.Sleep(r.Policy(context.Background(), i))
		}

		err = fn()
		if errors.Is(err, ErrAborted) {
			return err
		}

		if err == nil {
			return nil
		}
	}

	return errors.Join(ErrRetryLimitExceeded, err)
}

func (r *Retry) DoCtx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	for i := range r.Attempts {
		if i != 0 {
			time.Sleep(r.Policy(ctx, i))
		}

		err = fn(ctx)
		if errors.Is(err, ErrAborted) {
			return err
		}

		if err == nil {
			return nil
		}
	}

	return errors.Join(ErrRetryLimitExceeded, err)
}

func ExponentialBackoff(base, cap time.Duration, jitter bool) func(ctx context.Context, attempts int) time.Duration {
	b := float64(base)
	c := float64(cap)
	return func(ctx context.Context, attempts int) time.Duration {
		a := float64(attempts)
		j := 1.0
		if jitter {
			j = 1.0 + rand.Float64()
		}
		return time.Duration(min(c, j*b*math.Pow(2, a)))
	}
}
