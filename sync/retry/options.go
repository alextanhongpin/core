package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const defaultAttempts = 10

var (
	N      = WithAttempts
	NoWait = Constant(0)
)

type Options struct {
	Attempts  int
	Backoff   Backoff
	Throttler Limiter
	Retryable func(err error) (error, bool)
}

func NewOptions() *Options {
	return &Options{
		Attempts:  defaultAttempts,
		Backoff:   NewExponentialBackoff(time.Second, time.Minute),
		Throttler: NewNoOpThrottler(),
		Retryable: func(err error) (error, bool) {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("%w: %w", ErrCanceled, err), false
			}
			return err, true
		},
	}
}

type Option func(*Options)

func Constant(base time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewConstantBackoff(base)
	}
}

func Exponential(base, cap time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewExponentialBackoff(base, cap)
	}
}

func Linear(base time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewLinearBackoff(base)
	}
}

func Throttle() Option {
	return WithThrottler(NewThrottler(NewThrottlerOptions()))
}

func WithAttempts(n int) Option {
	if n < 0 {
		panic("attempts must be non-negative")
	}
	return func(o *Options) {
		o.Attempts = n
	}
}

func WithBackoff(bf Backoff) Option {
	return func(o *Options) {
		o.Backoff = bf
	}
}

func WithThrottler(t Limiter) Option {
	return func(o *Options) {
		o.Throttler = t
	}
}

func WithRetryable(fn func(err error) (error, bool)) Option {
	return func(o *Options) {
		o.Retryable = fn
	}
}

func WithNonRetryableErrors(errs ...error) Option {
	joinErr := errors.Join(errs...)
	return func(o *Options) {
		prev := o.Retryable
		o.Retryable = func(err error) (error, bool) {
			if errors.Is(joinErr, err) {
				return fmt.Errorf("%w: %w", ErrCanceled, err), false
			}
			if prev != nil {
				return prev(err)
			}
			return err, true
		}
	}
}
