// Package retry implements functions for retry mechanism.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

var ErrMaxAttempts = errors.New("retry: max attempts reached")

type BackoffType int

const (
	BackoffTypeExponential BackoffType = iota
	BackoffTypeLinear
	BackoffTypeConstant
)

type Option struct {
	BackoffType      BackoffType
	Delay            time.Duration
	MaxDelay         time.Duration
	MaxRetryAttempts int
	MaxDuration      time.Duration
	UseJitter        bool
}

func NewOption() *Option {
	return &Option{
		BackoffType:      BackoffTypeExponential,
		Delay:            100 * time.Millisecond,
		MaxDelay:         time.Minute,
		MaxRetryAttempts: 10,
		UseJitter:        true,
		MaxDuration:      5 * time.Minute,
	}
}

type Retry[T any] struct {
	// Returns a bool to indicate if a retry should be made,
	// and also the error for the decision.
	ShouldHandle func(T, error) (bool, error)
	OnRetry      func(Event)
	backoff      Backoff
	useJitter    bool
}

type Event struct {
	Attempt int
	Delay   time.Duration
	Err     error
}

func New[T any](opt *Option) *Retry[T] {
	if opt == nil {
		opt = NewOption()
	}

	backoff := NewBackoff(opt.BackoffType, opt.MaxRetryAttempts, opt.Delay).
		WithMaxDelay(opt.MaxDelay).
		WithMaxDuration(opt.MaxDuration).
		WithMaxRetryAttempts(opt.MaxRetryAttempts)

	return &Retry[T]{
		backoff:   backoff,
		useJitter: opt.UseJitter,
		ShouldHandle: func(v T, err error) (bool, error) {
			// Skip if cancelled by caller.
			if errors.Is(err, context.Canceled) {
				return false, err
			}

			return err != nil, err
		},
		OnRetry: func(Event) {},
	}
}

func (r *Retry[T]) Do(fn func() (T, error)) (v T, res Result, err error) {
	var shouldRetry bool

	backoff := r.backoff
	if r.useJitter {
		backoff = backoff.WithJitter()
	}

	// The first execution does not count as retry.
	backoff = append([]time.Duration{0}, backoff...)
	for i, t := range backoff {
		if i != 0 {
			res.Attempts = i
			res.Duration += t

			r.OnRetry(Event{
				Attempt: res.Attempts,
				Delay:   t,
				Err:     err,
			})
		}
		if t != 0 {
			time.Sleep(t)
		}

		v, err = fn()
		// We may not have an error, but the result is not what we want.
		// E.g. a HTTP request may SUCCEED with status code 5XX.
		shouldRetry, err = r.ShouldHandle(v, err)
		if !shouldRetry {
			return
		}
	}

	if shouldRetry {
		errMaxAttempts := fmt.Errorf("%w - %s", ErrMaxAttempts, res.String())
		if err != nil {
			err = fmt.Errorf("%w: %w", errMaxAttempts, err)
		} else {
			err = errMaxAttempts
		}
	}

	return
}

type Backoff []time.Duration

// NewBackoff generates a list of backoff durations based on the backoff type.
func NewBackoff(t BackoffType, n int, delay time.Duration) Backoff {
	res := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		switch t {
		case BackoffTypeLinear:
			res[i] = time.Duration(i+1) * delay
		case BackoffTypeExponential:
			res[i] = time.Duration(math.Pow(2, float64(i))) * delay
		default:
			// Defaults to constant.
			res[i] = delay
		}
	}

	return res
}

func (b Backoff) WithJitter() Backoff {
	return WithJitter(b)
}

func (b Backoff) WithMaxRetryAttempts(n int) Backoff {
	return WithMaxRetryAttempts(b, n)
}

func (b Backoff) WithMaxDelay(d time.Duration) Backoff {
	return WithMaxDelay(b, d)
}

func (b Backoff) WithMaxDuration(d time.Duration) Backoff {
	return WithMaxDuration(b, d)
}

// WithMaxRetryAttempts limits the number of attempts.
func WithMaxRetryAttempts(ts []time.Duration, n int) []time.Duration {
	if len(ts) <= n {
		return ts
	}

	return ts[:n]
}

// WithMaxDelay caps the delay to the max value.
func WithMaxDelay(ts []time.Duration, maxDelay time.Duration) []time.Duration {
	res := make([]time.Duration, len(ts))

	for i := range ts {
		res[i] = min(ts[i], maxDelay)
	}

	return res
}

// WithMaxDuration caps the total duration of the retry.
func WithMaxDuration(ts []time.Duration, maxDuration time.Duration) []time.Duration {
	res := make([]time.Duration, 0, len(ts))

	var total time.Duration
	for i := range ts {
		d := ts[i]
		total += d
		if total > maxDuration {
			return res
		}

		res = append(res, d)
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

type Result struct {
	Attempts int
	Duration time.Duration
}

func (r *Result) String() string {
	return fmt.Sprintf("retry %d times, took %s", r.Attempts, r.Duration)
}

func jitter(d time.Duration) time.Duration {
	if d == 0 {
		return 0
	}

	return time.Duration(Rand.Intn(int(d / 2))).Round(5 * time.Millisecond)
}
