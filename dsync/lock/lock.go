// Package lock provides distributed locking mechanisms using Redis.
//
// The package offers two main implementations:
//   - Basic Locker: Simple distributed locking with exponential backoff
//   - PubSub Locker: Optimized locking using Redis pub/sub for faster acquisition
//
// Key features:
//   - Automatic lock refresh during long operations
//   - Context-based cancellation and timeouts
//   - Configurable backoff strategies
//   - Keyed mutexes to prevent local deadlocks
//   - Comprehensive error handling
//
// Example usage:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	locker := lock.New(client)
//
//	err := locker.Do(ctx, "resource-key", func(ctx context.Context) error {
//		// Critical section
//		return nil
//	}, &lock.Config{
//		Lock: 30 * time.Second,
//		Wait: 10 * time.Second,
//		RefreshRatio: 0.8,
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
)

var (
	ErrExpired         = errors.New("lock: lock expired")
	ErrLockTimeout     = errors.New("lock: exceeded lock duration")
	ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
	ErrLocked          = errors.New("lock: another process has acquired the lock")
)

type Config struct {
	//  The duration to wait for the lock to be available.
	WaitTTL time.Duration
	// The duration for which the lock is held.
	LockTTL time.Duration
	// The ratio of the lock duration to refresh the lock.
	RefreshRatio float64
}

func (c *Config) Validate() error {
	if c.LockTTL == 0 {
		return errors.New("lock: lock duration cannot be zero")
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		WaitTTL:      5 * time.Second,
		LockTTL:      30 * time.Second,
		RefreshRatio: 0.8,
	}
}

// Locker represents a distributed lock implementation using Redis.
// Works on with a single redis node.
type Locker struct {
	*Config
	*cache.Cache[string, sync.Mutex]
	Logger *slog.Logger // Optional logger for debugging purposes.
	client
}

// New returns a pointer to Locker.
func New(c client, cfg *Config) *Locker {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	return &Locker{
		Config: cmp.Or(cfg, DefaultConfig()),
		Cache: cache.New[string, sync.Mutex](func(string) (*sync.Mutex, error) {
			return new(sync.Mutex), nil
		}),
		Logger: slog.Default(), // Default logger, can be overridden.
		client: c,
	}
}

func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error) error {
	mu, _ := l.Cache.Get(key)
	mu.Lock()
	defer mu.Unlock()

	token := uuid.NewV7().String()

	// Try to acquire the lock.
	if err := l.Lock(ctx, key, token, l.LockTTL, l.WaitTTL); err != nil {
		return err
	}

	unlock := sync.OnceValue(func() error {
		return l.Unlock(context.WithoutCancel(ctx), key, token)
	})
	// Lock acquired. Remember to unlock.
	defer func() {
		if err := unlock(); err != nil {
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
		go func() {
			ch <- fn(ctx)
			close(ch)
		}()

		select {
		// In case the client does not handle context cancellation.
		case <-ctx.Done():
			return errors.Join(context.Cause(ctx), unlock())

		case err := <-ch:
			return errors.Join(err, unlock())
		}
	}

	// Create a channel with a buffer of 1 to prevent goroutine leak.
	ch := make(chan error, 1)

	go func() {
		ch <- fn(ctx)
		close(ch)
	}()

	t := time.NewTicker(refresh)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.Join(context.Cause(ctx), unlock())

		case err := <-ch:
			return errors.Join(err, unlock())

		case <-t.C:
			if err := l.Extend(ctx, key, token, l.LockTTL); err != nil {
				return errors.Join(err, unlock())
			}
		}
	}
}
