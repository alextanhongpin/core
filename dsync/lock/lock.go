// Package lock provides renewable, token-checked leases on one Redis primary.
package lock

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"
	"uuid"

	"github.com/alextanhongpin/core/sync/cache"
	"github.com/alextanhongpin/core/sync/retry"
)

var (
	ErrExpired     = errors.New("lock: lock expired")
	ErrLockTimeout = errors.New("lock: exceeded lock duration")
	ErrLocked      = errors.New("lock: another process has acquired the lock")
)

type Retry interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Config struct {
	// The duration for which the lock is held.
	LockTTL time.Duration
	// The ratio of the lock duration to refresh the lock.
	RefreshRatio float64

	// The retry for acquiring lock.
	Retry  Retry
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.LockTTL < time.Millisecond {
		return errors.New("lock: lock duration must be greater than zero")
	}
	if math.IsNaN(c.RefreshRatio) || math.IsInf(c.RefreshRatio, 0) || c.RefreshRatio < 0 || c.RefreshRatio >= 1 {
		return errors.New("lock: refresh ratio must be in the range [0, 1)")
	}

	if c.RefreshRatio > 0 && time.Duration(float64(c.LockTTL)*c.RefreshRatio) <= 0 {
		return errors.New("lock: refresh interval must be positive")
	}
	return nil
}

func DefaultRetry() Retry {
	cfg := retry.DefaultConfig()
	cfg.MaxRetries = 10
	cfg.Backoff = retry.NewExponentialBackoff(50*time.Millisecond, 2*time.Second)
	cfg.Throttler = retry.NewNoopThrottler()
	cfg.Retryable = func(err error) (error, bool) { return err, errors.Is(err, ErrLocked) }
	r, _ := retry.New(cfg)
	return r
}

func DefaultConfig() Config {
	return Config{
		LockTTL:      30 * time.Second,
		RefreshRatio: 0.8,
		Retry:        DefaultRetry(),
	}
}

// WithDefaults fills omitted dependencies and TTL. Zero RefreshRatio disables
// renewal; DefaultConfig explicitly enables it. Dependencies remain shared.
func (c Config) WithDefaults() Config {
	if c.LockTTL == 0 {
		c.LockTTL = 30 * time.Second
	}
	if c.Retry == nil {
		c.Retry = DefaultRetry()
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

type permit struct{ ch chan struct{} }

// Locker owns configuration and local admission state. Callbacks and client
// dependencies may be used concurrently. Construct with New.
type Locker struct {
	cfg     Config
	permits *cache.Cache[string, permit]
	client
}

func New(c client, cfg Config) (*Locker, error) {
	if c == nil {
		return nil, errors.New("lock: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Locker{cfg: cfg, client: c, permits: cache.New(func(string) (*permit, error) { return &permit{ch: make(chan struct{}, 1)}, nil })}, nil
}

func MustNew(c client, cfg Config) *Locker {
	l, err := New(c, cfg)
	if err != nil {
		panic(err)
	}
	return l
}

// Do executes fn synchronously. Cancellation and lease loss cancel the callback
// context, but Do waits for fn to finish before releasing ownership. Callbacks
// must cooperate with cancellation and must not reenter the same key.
func (l *Locker) Do(ctx context.Context, key string, fn func(context.Context) error) (err error) {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if key == "" || fn == nil {
		return errors.New("lock: key and callback are required")
	}
	p, _, _ := l.permits.LoadOrCreate(key)
	select {
	case p.ch <- struct{}{}:
	case <-ctx.Done():
		return context.Cause(ctx)
	}
	defer func() { <-p.ch }()
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	token := uuid.NewV7().String()
	if err := l.cfg.Retry.Do(ctx, func(ctx context.Context) error { return l.Lock(ctx, key, token, l.cfg.LockTTL) }); err != nil {
		return err
	}
	workCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	refresh := time.Duration(float64(l.cfg.LockTTL) * l.cfg.RefreshRatio)
	if refresh <= 0 {
		timeoutCtx, timeoutCancel := context.WithTimeoutCause(workCtx, l.cfg.LockTTL, ErrLockTimeout)
		defer timeoutCancel()
		workCtx = timeoutCtx
	}
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		if refresh <= 0 {
			return
		}
		ticker := time.NewTicker(refresh)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-workCtx.Done():
				return
			case <-ticker.C:
				if err := l.Extend(workCtx, key, token, l.cfg.LockTTL); err != nil {
					cancel(err)
					return
				}
			}
		}
	}()
	defer func() {
		close(stop)
		<-stopped
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cleanupCancel()
		err = errors.Join(err, l.Unlock(cleanupCtx, key, token))
	}()
	err = fn(workCtx)
	if workCtx.Err() != nil {
		err = errors.Join(err, context.Cause(workCtx))
	}
	return err
}
