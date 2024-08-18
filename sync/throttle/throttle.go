package throttle

import (
	"cmp"
	"context"
	"errors"
	"time"
)

var (
	ErrTimeout          = errors.New("throttle: timeout")
	ErrCapacityExceeded = errors.New("throttle: capacity exceeded")
)

type Options struct {
	BacklogLimit   int
	BacklogTimeout time.Duration
	Limit          int
}

func NewOptions() *Options {
	return &Options{
		Limit:          1000,
		BacklogLimit:   100,
		BacklogTimeout: 10 * time.Second,
	}
}

func (o *Options) Valid() error {
	if o.Limit <= 0 {
		return errors.New("throttle: limit must be greater than 0")
	}

	if o.BacklogLimit < 0 {
		return errors.New("throttle: backlog limit must be greater or equal to 0")
	}

	if o.BacklogTimeout < 0 {
		return errors.New("throttle: backlog timeout must be greater or equal to 0")
	}

	return nil
}

type Throttler struct {
	ch        chan struct{}
	backlogCh chan struct{}
	opts      *Options
}

func New(opts *Options) *Throttler {
	opts = cmp.Or(opts, NewOptions())
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	ch := make(chan struct{}, opts.Limit)
	backlogCh := make(chan struct{}, opts.Limit+opts.BacklogLimit)
	for i := range opts.Limit + opts.BacklogLimit {
		if i < opts.Limit {
			ch <- struct{}{}
		}
		backlogCh <- struct{}{}
	}

	return &Throttler{
		opts:      opts,
		ch:        ch,
		backlogCh: backlogCh,
	}
}

func (t *Throttler) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeoutCause(ctx, t.opts.BacklogTimeout, ErrTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-t.backlogCh:
		defer func() {
			t.backlogCh <- struct{}{}
		}()

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-t.ch:
			defer func() {
				t.ch <- struct{}{}
			}()
			return fn(ctx)
		}
	default:
		return ErrCapacityExceeded
	}
}
