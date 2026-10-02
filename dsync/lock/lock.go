// Package lock provides a distributed locking mechanism using Redis.
//
// Key features:
//   - Atomic acquisition and release using Redis SETNX and DELEX/SETIFDEQ
//   - Automatic lock refresh during long operations via RefreshRatio
//   - Configurable retry strategies via sync/retry
//   - Context-based cancellation and timeouts
//   - In-process keyed mutexes to serialize concurrent attempts within the same process
//   - Safe panic recovery to guarantee lock release
//
// Example usage:
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	client := lock.NewClient(rdb)
//	locker := lock.New(client, &lock.Config{
//		LockTTL:      30 * time.Second,
//		RefreshRatio: 0.8,
//	})
//
//	err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
//		// Critical section
//		return nil
//	})
package lock

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"sync"
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
	Retry Retry
}

func (c *Config) Validate() error {
	if c.LockTTL <= 0 {
		return errors.New("lock: lock duration must be greater than zero")
	}
	if c.RefreshRatio < 0 || c.RefreshRatio >= 1 {
		return errors.New("lock: refresh ratio must be in the range [0, 1)")
	}

	return nil
}

func DefaultRetry() Retry {
	cfg := retry.DefaultConfig()
	cfg.Attempts = 10
	cfg.Backoff = retry.NewExponentialBackoff(50*time.Millisecond, 2*time.Second)
	cfg.Throttler = retry.NewNoopThrottler()
	return retry.New(cfg)
}

func DefaultConfig() *Config {
	return &Config{
		LockTTL:      30 * time.Second,
		RefreshRatio: 0.8,
		Retry:        DefaultRetry(),
	}
}

// Locker represents a distributed lock implementation using Redis.
// Works on a single redis node.
type Locker struct {
	*Config
	*cache.Cache[string, sync.Mutex]
	Logger *slog.Logger // Optional logger for debugging purposes.
	client
}

// New returns a pointer to Locker.
func New(c client, cfg *Config) *Locker {
	if c == nil {
		panic(errors.New("lock: client cannot be nil"))
	}
	cfg = cmp.Or(cfg, DefaultConfig())
	if cfg.Retry == nil {
		cfg.Retry = DefaultRetry()
	}
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	return &Locker{
		Config: cfg,
		Cache: cache.New(func(string) (*sync.Mutex, error) {
			return new(sync.Mutex), nil
		}),
		Logger: slog.Default(), // Default logger, can be overridden.
		client: c,
	}
}

func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error) error {
	mu, _, _ := l.Cache.LoadOrCreate(key)
	mu.Lock()
	defer mu.Unlock()

	token := uuid.NewV7().String()

	// Try to acquire the lock.
	if err := l.Config.Retry.Do(ctx, func(ctx context.Context) error {
		return l.Lock(ctx, key, token, l.LockTTL)
	}); err != nil {
		return err
	}

	unlock := sync.OnceValue(func() error {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return l.Unlock(unlockCtx, key, token)
	})
	// Lock acquired. Remember to unlock.
	defer func() {
		if err := unlock(); err != nil && !errors.Is(err, ErrExpired) {
			l.Logger.Error("unlocking", "key", key, "token", token, "err", err)
		}
	}()

	// No refresh.
	refresh := time.Duration(float64(l.LockTTL) * l.RefreshRatio)
	if refresh <= 0 {
		// Strictly no refresh, the operation will timeout with error.
		ctx, cancel := context.WithTimeoutCause(ctx, l.LockTTL, ErrLockTimeout)
		defer cancel()

		ch := make(chan error, 1)
		panicVal := make(chan any, 1)

		go func() {
			defer close(ch)
			defer func() {
				if r := recover(); r != nil {
					panicVal <- r
				}
			}()
			ch <- fn(ctx)
		}()

		select {
		case p := <-panicVal:
			_ = unlock()
			panic(p)

		case <-ctx.Done():
			return errors.Join(context.Cause(ctx), unlock())

		case err := <-ch:
			if ctx.Err() != nil {
				return errors.Join(context.Cause(ctx), err, unlock())
			}
			return errors.Join(err, unlock())
		}
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	ch := make(chan error, 1)
	panicVal := make(chan any, 1)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				panicVal <- r
			}
		}()
		ch <- fn(ctx)
	}()

	t := time.NewTicker(refresh)
	defer t.Stop()

	for {
		select {
		case p := <-panicVal:
			_ = unlock()
			panic(p)

		case <-ctx.Done():
			return errors.Join(context.Cause(ctx), unlock())

		case err := <-ch:
			if ctx.Err() != nil {
				return errors.Join(context.Cause(ctx), err, unlock())
			}
			return errors.Join(err, unlock())

		case <-t.C:
			if err := l.Extend(ctx, key, token, l.LockTTL); err != nil {
				cancel(err)
				return errors.Join(err, unlock())
			}
		}
	}
}
