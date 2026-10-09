package throttle

import (
	"context"
	"errors"
	"math"
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
func DefaultConfig() Config {
	return Config{
		Limit:          1000,
		BacklogLimit:   100,
		BacklogTimeout: 10 * time.Second,
	}
}

// WithDefaults fills the concurrency limit. Zero backlog disables queueing and
// zero timeout disables waiting; use DefaultConfig for a queued configuration.
func (c Config) WithDefaults() Config {
	if c.Limit == 0 {
		c.Limit = DefaultConfig().Limit
	}
	return c
}

// Validate checks if the Config settings are valid.
func (c Config) Validate() error {
	if c.Limit <= 0 {
		return errors.New("throttle: limit must be greater than 0")
	}

	if c.BacklogLimit < 0 {
		return errors.New("throttle: backlog limit must be greater or equal to 0")
	}

	if c.BacklogTimeout < 0 {
		return errors.New("throttle: backlog timeout must be greater or equal to 0")
	}
	if c.BacklogLimit > math.MaxInt-c.Limit {
		return errors.New("throttle: total capacity overflows int")
	}
	return nil
}

// Throttler manages the throttling logic using channel buffers.
type Throttler struct {
	ch        chan struct{}
	backlogCh chan struct{}
	cfg       Config
}

// New creates and initializes a new Throttler.
func New(cfg Config) (*Throttler, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
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
		cfg:       cfg,
	}, nil
}

// MustNew is New for startup wiring that must panic on invalid configuration.
func MustNew(cfg Config) *Throttler {
	t, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Do executes fn with concurrency throttling. BacklogTimeout only limits
// admission waiting; fn receives the original caller context. A zero timeout
// allows immediate admission but does not wait for a busy slot.
// Configuration is privately owned. Callbacks may run concurrently.
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
		waitCtx, cancel := context.WithTimeoutCause(ctx, t.cfg.BacklogTimeout, ErrTimeout)
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
