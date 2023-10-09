// Package retry implements functions for retry mechanism.
package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

var (
	Linear      = Backoffs(linear())
	Exponential = Backoffs(exponential())
)

type Backoffs []time.Duration

func (b Backoffs) Jitter() Backoffs {
	return WithJitter(b)
}

func (b Backoffs) SoftLimit(limit time.Duration) Backoffs {
	return WithSoftLimit(b, limit)
}

func (b Backoffs) HardLimit(limit time.Duration) Backoffs {
	return WithHardLimit(b, limit)
}

func (b Backoffs) Exec(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return Exec(ctx, fn, b)
}

func (b Backoffs) ExecResult(ctx context.Context, fn func(ctx context.Context) error) (res Result, err error) {
	return ExecResult(ctx, fn, b)
}

// WithSoftLimit applies soft limit to the total duration. The total duration
// will be at least the soft limit amount.
func WithSoftLimit(ts []time.Duration, limit time.Duration) []time.Duration {
	res := make([]time.Duration, 0, len(ts))

	var total time.Duration
	for i := range ts {
		total += ts[i]
		res = append(res, ts[i])

		if total > limit {
			break
		}
	}

	return res
}

// WithHardLimit applies hard limit to the total duration. The total duration
// must be at most the hard limit amount.
func WithHardLimit(ts []time.Duration, limit time.Duration) []time.Duration {
	res := make([]time.Duration, 0, len(ts))

	var total time.Duration
	for i := range ts {
		total += ts[i]
		if total > limit {
			allowed := ts[i] - (total - limit)
			res = append(res, allowed)
			break
		}

		res = append(res, ts[i])
	}

	return res
}

// WithJitter includes jitter to each duration.
func WithJitter(ts []time.Duration) []time.Duration {
	res := make([]time.Duration, len(ts))
	for i := range ts {
		res[i] = jitter(ts[i]) + ts[i]
	}

	return res
}

// Exec executes the retry and returns an error.
func Exec(ctx context.Context, fn func(ctx context.Context) error, ts []time.Duration) (err error) {
	timeouts := append([]time.Duration{0}, ts...)

	for _, t := range timeouts {
		if t != 0 {
			time.Sleep(t)
		}

		err = fn(ctx)
		if err == nil {
			return
		}

	}

	return
}

type Result struct {
	Skip  bool
	Retry int
}

// ExecResult executes the retry and returns an error and result.
func ExecResult(ctx context.Context, fn func(ctx context.Context) error, ts []time.Duration) (res Result, err error) {
	timeouts := append([]time.Duration{0}, ts...)

	for i, t := range timeouts {
		if t != 0 {
			time.Sleep(t)
		}

		err = fn(ctx)
		if err == nil {
			return
		}

		res.Retry = i

		var skipErr *SkipError
		if errors.As(err, &skipErr) {
			res.Skip = true
			err = skipErr.Unwrap()

			return
		}
	}

	return
}

// Do executes the retry and returns an error and result.
func Do[T any](ctx context.Context, fn func(ctx context.Context) (T, error), ts []time.Duration) (v T, err error, res Result) {
	timeouts := append([]time.Duration{0}, ts...)

	for i, t := range timeouts {
		if t != 0 {
			time.Sleep(t)
		}

		v, err = fn(ctx)
		if err == nil {
			return
		}

		res.Retry = i

		var skipErr *SkipError
		if errors.As(err, &skipErr) {
			res.Skip = true
			err = skipErr.Unwrap()

			return
		}
	}

	return
}

func linear() []time.Duration {
	n := 10

	res := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		res[i] = 1 * time.Second
	}

	return res
}

func exponential() []time.Duration {
	n := 10

	res := make([]time.Duration, n)
	res[0] = 50 * time.Millisecond
	for i := 1; i < n; i++ {
		res[i] = res[i-1] * 2
	}

	return res
}

func jitter(d time.Duration) time.Duration {
	return time.Duration(rand.Intn(int(d / 2))).Round(5 * time.Millisecond)
}
