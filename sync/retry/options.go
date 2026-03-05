package retry

import "time"

var (
	N        = WithAttempts
	NoWait   = Constant(0)
	Disabled = WithAttempts(0)
)

func Throttle() Option {
	return WithThrottler(NewThrottler(NewThrottlerOptions()))
}

type Options struct {
	Backoff          backoff
	Throttler        throttler
	Attempts         int
	MetricsCollector RetryMetricsCollector
}

func (o *Options) Clone() *Options {
	return &Options{
		Backoff:          o.Backoff,
		Throttler:        o.Throttler,
		Attempts:         o.Attempts,
		MetricsCollector: o.MetricsCollector,
	}
}

func (o *Options) With(opts ...Option) *Options {
	return o.Clone().Apply(opts...)
}

func (o *Options) Apply(opts ...Option) *Options {
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func NewOptions() *Options {
	return &Options{
		Backoff:          NewExponentialBackoff(time.Second, time.Minute),
		Throttler:        NewNoOpThrottler(),
		Attempts:         10,
		MetricsCollector: &AtomicRetryMetricsCollector{},
	}
}

type Option func(*Options)

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

func WithMetricsCollector(mc RetryMetricsCollector) Option {
	return func(o *Options) {
		o.MetricsCollector = mc
	}
}

func Exponential(base, cap time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewExponentialBackoff(base, cap)
	}
}

func Constant(base time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewConstantBackoff(base)
	}
}

func Linear(base time.Duration) Option {
	return func(o *Options) {
		o.Backoff = NewLinearBackoff(base)
	}
}
