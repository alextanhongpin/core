// Package retry implements functions for DoFunc mechanism.
package retry

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrAborted       = errors.New("retry: aborted")
)

type PolicyFunc func(i int) time.Duration

type Options struct {
	Attempts int
	Policy   PolicyFunc
}

func NewOptions() *Options {
	return &Options{
		Attempts: 10,
		Policy:   ExponentialBackoff(100*time.Millisecond, 1*time.Minute, true),
	}
}

type Handler struct {
	opts *Options
}

func New(opts *Options) *Handler {
	if opts == nil {
		opts = NewOptions()
	}
	if opts.Attempts < 1 {
		panic("retry: attempts must be greater than 0")
	}
	if opts.Policy == nil {
		panic("retry: policy must be set")
	}

	return &Handler{
		opts: opts,
	}
}

func (r *Handler) Do(fn func() error) error {
	return DoFunc(fn, WithAttempts(r.opts.Attempts), WithPolicy(r.opts.Policy))
}

type Option func(opts *Options)

func WithAttempts(attempts int) Option {
	return func(opts *Options) {
		opts.Attempts = attempts
	}
}

func WithPolicy(policy PolicyFunc) Option {
	return func(opts *Options) {
		opts.Policy = policy
	}
}

func DoFunc(fn func() error, opts ...Option) (err error) {
	o := NewOptions()
	for _, opt := range opts {
		opt(o)
	}

	for i := range o.Attempts {
		time.Sleep(o.Policy(i))

		err = fn()
		if err == nil {
			return nil
		}

		if errors.Is(err, ErrAborted) {
			return err
		}
	}

	return limitExceeded(err)
}

func DoFunc2[T any](fn func() (T, error), opts ...Option) (res T, err error) {
	o := NewOptions()
	for _, opt := range opts {
		opt(o)
	}

	for i := range o.Attempts {
		time.Sleep(o.Policy(i))

		res, err = fn()
		if err == nil {
			return res, nil
		}

		if errors.Is(err, ErrAborted) {
			return res, err
		}
	}

	return res, limitExceeded(err)
}

func Abort(err error) error {
	return &retryError{
		err: ErrAborted,
		ori: err,
	}
}

func limitExceeded(err error) error {
	return &retryError{
		err: ErrLimitExceeded,
		ori: err,
	}
}

func ExponentialBackoff(base, cap time.Duration, jitter bool) func(attempts int) time.Duration {
	b := float64(base)
	c := float64(cap)

	return func(attempts int) time.Duration {
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

type retryError struct {
	err error
	ori error
}

func (t *retryError) Error() string {
	return fmt.Sprintf("%s: %s", t.err, t.ori)
}

func (t *retryError) Unwrap() error {
	return t.ori
}

func (t *retryError) Is(err error) bool {
	return errors.Is(t.err, err) || errors.Is(t.ori, err)
}
