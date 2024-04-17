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

var (
	// ErrMaxAttempts is returned when the max attempts is reached.
	ErrMaxAttempts = errors.New("retry: max attempts reached")

	// ErrWaitTimeout is returned when the retry still failed after the wait timeout.
	ErrWaitTimeout = errors.New("retry: wait timeout exceeded")
)

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

type Retry struct {
	// Returns a bool to indicate if a retry should be made,
	// and also the error for the decision.
	OnRetry     func(Event)
	backoff     Backoff
	useJitter   bool
	maxDuration time.Duration
}

type Event struct {
	Attempt int
	Delay   time.Duration
	Err     error
}

func New(opt *Option) *Retry {
	if opt == nil {
		opt = NewOption()
	}

	backoff := NewBackoff(opt.BackoffType, opt.MaxRetryAttempts, opt.Delay).
		WithMaxDelay(opt.MaxDelay).
		WithMaxRetryAttempts(opt.MaxRetryAttempts)

	return &Retry{
		backoff:     backoff,
		useJitter:   opt.UseJitter,
		maxDuration: opt.MaxDuration,
		OnRetry:     func(Event) {},
	}
}

func (r *Retry) Do(ctx context.Context, fn func(ctx context.Context) error) (res *Result, err error) {
	backoff := r.backoff
	if r.useJitter {
		backoff = backoff.WithJitter()
	}

	ctx, cancel := context.WithTimeoutCause(ctx, r.maxDuration, ErrWaitTimeout)
	defer cancel()

	res = new(Result)

	// The first execution does not count as retry.
	backoff = append([]time.Duration{0}, backoff...)
	for i, t := range backoff {
		select {
		case <-ctx.Done():
			return res, context.Cause(ctx)
		default:
			time.Sleep(t)
		}

		if i != 0 {
			res.Attempts = i
			res.Duration += t

			// Useful for recording metrics.
			r.OnRetry(Event{
				Attempt: res.Attempts,
				Delay:   t,
				Err:     err,
			})
		}

		err = fn(ctx)
		if err == nil {
			return
		}
	}

	return nil, errors.Join(ErrMaxAttempts, err)
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
