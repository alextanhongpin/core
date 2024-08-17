// Package retry implements functions for DoFunc mechanism.
package retry

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrAborted       = errors.New("retry: aborted")
	ErrThrottled     = errors.New("retry: throttled")
)

type PolicyFunc func(i int) time.Duration

type Options struct {
	Attempts  int
	Policy    PolicyFunc
	Throttler *Throttler
}

func NewOptions() *Options {
	return &Options{
		Attempts:  10,
		Policy:    ExponentialBackoff(100*time.Millisecond, 1*time.Minute, true),
		Throttler: NewThrottler(NewThrottlerOptions()),
	}
}

func (o *Options) Valid() error {
	if o.Attempts < 1 {
		return errors.New("retry: attempts must be greater than 0")
	}
	if o.Policy == nil {
		return errors.New("retry: policy must be set")
	}

	return nil
}

type Handler struct {
	opts *Options
}

func New(opts *Options) *Handler {
	opts = cmp.Or(opts, NewOptions())
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	return &Handler{
		opts: opts,
	}
}

func (r *Handler) Do(fn func() error) error {
	return DoFunc(fn,
		WithAttempts(r.opts.Attempts),
		WithPolicy(r.opts.Policy),
		WithThrottler(r.opts.Throttler),
	)
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

func WithThrottler(t *Throttler) Option {
	return func(opts *Options) {
		opts.Throttler = t
	}
}

func DoFunc(fn func() error, opts ...Option) (err error) {
	o := NewOptions()
	for _, opt := range opts {
		opt(o)
	}

	for i := range o.Attempts {
		t := o.Throttler

		time.Sleep(o.Policy(i))
		if i > 0 && !t.allow() {
			return Abort(ErrThrottled)
		}

		err = fn()
		if err == nil {
			t.success()

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
		t := o.Throttler
		time.Sleep(o.Policy(i))
		if i > 0 && !t.allow() {
			return res, Abort(ErrThrottled)
		}

		res, err = fn()
		if err == nil {
			t.success()

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

type Throttler struct {
	ratio  float64
	thresh float64 // max / 2
	max    float64

	mu     sync.Mutex
	tokens float64
}

type ThrottlerOptions struct {
	MaxTokens  float64
	TokenRatio float64
}

func NewThrottlerOptions() *ThrottlerOptions {
	return &ThrottlerOptions{
		MaxTokens:  10,
		TokenRatio: 0.1,
	}
}

func NewThrottler(opts *ThrottlerOptions) *Throttler {
	opts = cmp.Or(opts, NewThrottlerOptions())

	ratio := opts.TokenRatio
	maxTokens := opts.MaxTokens

	return &Throttler{
		ratio:  ratio,
		max:    maxTokens,
		tokens: maxTokens,
		thresh: maxTokens / 2,
	}
}

func (t *Throttler) allow() bool {
	if t == nil {
		return true
	}

	t.mu.Lock()

	t.tokens = max(t.tokens-1, 0)
	ok := t.tokens > t.thresh
	t.mu.Unlock()

	return ok
}

func (t *Throttler) success() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.tokens = min(t.tokens+t.ratio, t.max)
	t.mu.Unlock()
}
