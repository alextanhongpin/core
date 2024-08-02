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
		time.Sleep(r.Policy(context.Background(), i))

		err = fn()
		if err == nil {
			return nil
		}

		var abortErr *AbortError
		if errors.As(err, &abortErr) {
			return abortErr.Unwrap()
		}
	}

	return errors.Join(ErrRetryLimitExceeded, err)
}

func (r *Retry) DoCtx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	for i := range r.Attempts {
		time.Sleep(r.Policy(ctx, i))

		err = fn(ctx)
		if err == nil {
			return nil
		}

		var abortErr *AbortError
		if errors.As(err, &abortErr) {
			return abortErr.Unwrap()
		}
	}

	return errors.Join(ErrRetryLimitExceeded, err)
}

func ExponentialBackoff(base, cap time.Duration, jitter bool) func(ctx context.Context, attempts int) time.Duration {
	b := float64(base)
	c := float64(cap)

	return func(ctx context.Context, attempts int) time.Duration {
		if attempts <= 0 {
			return 0
		}

		a := float64(attempts)
		j := 1.0
		if jitter {
			j += rand.Float64()
		}
		e := math.Pow(2, a)

		return time.Duration(min(c, j*b*e))
	}
}

func Abort(err error) *AbortError {
	return &AbortError{err}
}

type AbortError struct {
	err error
}

func (e *AbortError) Error() string {
	return e.err.Error()
}

func (e *AbortError) Unwrap() error {
	return e.err
}
