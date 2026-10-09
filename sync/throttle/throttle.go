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

// Config holds the runtime configuration for the Throttler.
type Config struct {
	BacklogLimit   int
	BacklogTimeout time.Duration
	Limit          int
}

// DefaultConfig creates a default, valid Config.
func DefaultConfig() *Config {
	return &Config{
		Limit:          1000,
		BacklogLimit:   100,
		BacklogTimeout: 10 * time.Second,
	}
}

// Validate checks if the Config settings are valid.
func (c *Config) Validate() error {
	if c.Limit <= 0 {
		return errors.New("throttle: limit must be greater than 0")
	}

	if c.BacklogLimit < 0 {
		return errors.New("throttle: backlog limit must be greater or equal to 0")
	}

	if c.BacklogTimeout < 0 {
		return errors.New("throttle: backlog timeout must be greater or equal to 0")
	}
	return nil
}

// Throttler manages the throttling logic using channel buffers.
type Throttler struct {
	ch        chan struct{}
	backlogCh chan struct{}
	*Config
}

// New creates and initializes a new Throttler.
func New(cfg *Config) *Throttler {
	cfg = cmp.Or(cfg, DefaultConfig())
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	limit := cfg.Limit
	backlogLimit := cfg.BacklogLimit

	ch := make(chan struct{}, limit)
	backlogCh := make(chan struct{}, limit+backlogLimit)

	for range limit {
		ch <- struct{}{}
		backlogCh <- struct{}{}
	}
	for range backlogLimit {
		backlogCh <- struct{}{}
	}

	return &Throttler{
		ch:        ch,
		backlogCh: backlogCh,
		Config:    cfg,
	}
}

// Do executes fn with concurrency throttling. BacklogTimeout only limits
// admission waiting; fn receives the original caller context. A zero timeout
// allows immediate admission but does not wait for a busy slot.
// Config fields must not be changed concurrently with Do.
func (t *Throttler) Do(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	select {
	case <-t.backlogCh:
		defer func() { t.backlogCh <- struct{}{} }()
	default:
		return ErrCapacityExceeded
	}

	select {
	case <-t.ch:
	default:
		waitCtx, cancel := context.WithTimeoutCause(ctx, t.BacklogTimeout, ErrTimeout)
		defer cancel()
		select {
		case <-waitCtx.Done():
			return context.Cause(waitCtx)
		case <-t.ch:
		}
	}
	defer func() { t.ch <- struct{}{} }()
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	return fn(ctx)
}
