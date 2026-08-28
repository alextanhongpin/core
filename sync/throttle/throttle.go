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

// NewConfig creates a default, valid Config.
func NewConfig() *Config {
	return &Config{
		Limit:          1000,
		BacklogLimit:   100,
		BacklogTimeout: 10 * time.Second,
	}
}

type Option func(*Config)

func WithLimit(limit int) Option {
	return func(cfg *Config) {
		cfg.Limit = limit
	}
}

func WithBacklogLimit(limit int) Option {
	return func(cfg *Config) {
		cfg.BacklogLimit = limit
	}
}

func WithBacklogTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.BacklogTimeout = timeout
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
	cfg = cmp.Or(cfg, NewConfig())
	if cfg.Limit <= 0 {
		panic(errors.New("throttle: limit must be greater than 0"))
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

// Throttle attempts to acquire a token within the specified context timeout.
// It returns nil on success, ErrTimeout on context expiration, ErrCapacityExceeded when the limit is hit.
func (t *Throttler) Do(ctx context.Context, fn func(context.Context) error) error {
	// Set timeout context based on configuration
	ctx, cancel := context.WithTimeoutCause(ctx, t.BacklogTimeout, ErrTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return context.Cause(ctx)

	case <-t.backlogCh:
		// Try to acquire from backlog
		defer func() {
			select {
			case t.backlogCh <- struct{}{}:
			default:
			}
		}()

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-t.ch:
			// Acquired from primary channel
			defer func() {
				select {
				case t.ch <- struct{}{}:
				default:
				}
			}()
			return fn(ctx)
		}
	default:
		return ErrCapacityExceeded
	}
}
