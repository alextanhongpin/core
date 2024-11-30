package lock

import "time"

type Options struct {
	LockTTL time.Duration // How long the lock is held.
	WaitTTL time.Duration // How long to wait for the lock to be acquired.
}

func (o *Options) Apply(opts ...Option) *Options {
	for _, opt := range opts {
		opt(o)
	}

	return o
}

type Option func(o *Options)

func NoWait() Option {
	return func(o *Options) {
		o.WaitTTL = 0
	}
}

func WithLockTTL(t time.Duration) Option {
	return func(o *Options) {
		o.LockTTL = t
	}
}

func WithWaitTTL(t time.Duration) Option {
	return func(o *Options) {
		o.WaitTTL = t
	}
}
