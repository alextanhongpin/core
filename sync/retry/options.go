package retry

import "time"

const defaultAttempts = 10

var (
	N      = WithAttempts
	NoWait = Constant(0)
)

type Options struct {
	Attempts  int
	Backoff   backoff
	Throttler throttler
}

func NewOptions() *Options {
	return &Options{
		Attempts:  defaultAttempts,
		Backoff:   NewExponentialBackoff(time.Second, time.Minute),
		Throttler: NewNoOpThrottler(),
	}
}

func OptionsFrom(opts ...Option) *Options {
	o := NewOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
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
		panic("attempts must be greater than 0")
	}
	return func(o *Options) {
		o.Attempts = n
	}
}

func WithBackoff(bf backoff) Option {
	return func(o *Options) {
		o.Backoff = bf
	}
}

func WithThrottler(t throttler) Option {
	return func(o *Options) {
		o.Throttler = t
	}
}
