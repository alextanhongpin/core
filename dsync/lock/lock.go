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
//	}, &lock.LockOption{
//		Lock: 30 * time.Second,
//		Wait: 10 * time.Second,
//		RefreshRatio: 0.8,
//	})
package lock

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

var (
	ErrExpired         = errors.New("lock: lock expired")
	ErrLockTimeout     = errors.New("lock: exceeded lock duration")
	ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
	ErrLocked          = errors.New("lock: another process has acquired the lock")
)

type LockOption struct {
	//  The duration to wait for the lock to be available.
	Wait time.Duration
	// The duration for which the lock is held.
	Lock time.Duration
	// The ratio of the lock duration to refresh the lock.
	RefreshRatio float64
	// Optional token to identify the lock owner.
	Token string
}

func NewLockOption() *LockOption {
	return &LockOption{
		Wait:         5 * time.Second,
		Lock:         30 * time.Second,
		RefreshRatio: 0.8,
	}
}

// Locker represents a distributed lock implementation using Redis.
// Works on with a single redis node.
type Locker struct {
	Client  *redis.Client
	KeyLock *lock.KeyLock
	Logger  *slog.Logger // Optional logger for debugging purposes.
}

// New returns a pointer to Locker.
func New(client *redis.Client) *Locker {
	return &Locker{
		Client:  client,
		KeyLock: lock.New(),
		Logger:  slog.Default(), // Default logger, can be overridden.
	}
}

func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts *LockOption) error {
	locker := l.KeyLock.Lock(key)
	defer locker.Unlock()

	opts = cmp.Or(opts, NewLockOption())
	token := cmp.Or(opts.Token, newToken())
	if opts.Lock <= 0 {
		return fmt.Errorf("lock duration must be greater than zero, got %v", opts.Lock)
	}

	// Try to acquire the lock.
	if err := l.Lock(ctx, key, token, opts.Lock, opts.Wait); err != nil {
		return err
	}

	unlockOnce := sync.OnceValue(func() error {
		return l.Unlock(context.WithoutCancel(ctx), key, token)
	})
	// Lock acquired. Remember to unlock.
	defer func() {
		if unlockErr := unlockOnce(); unlockErr != nil {
			l.Logger.Error("failed to unlock", "key", key, "err", unlockErr)
		}
	}()

	// No refresh.
	if opts.RefreshRatio <= 0 {
		// Strictly no refresh, the operation will timeout with error.
		ctx, cancel := context.WithTimeoutCause(ctx, opts.Lock, ErrLockTimeout)
		defer cancel()

		ch := make(chan error, 1)
		go func() {
			ch <- errors.Join(fn(ctx), unlockOnce())
			close(ch)
		}()

		select {
		case <-ctx.Done():
			return context.Cause(ctx)

		case err := <-ch:
			return err
		}
	}

	// Create a channel with a buffer of 1 to prevent goroutine leak.
	ch := make(chan error, 1)

	go func() {
		ch <- errors.Join(fn(ctx), unlockOnce())
		close(ch)
	}()

	t := time.NewTicker(time.Duration(float64(opts.Lock) * opts.RefreshRatio))
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)

		case err := <-ch:
			return err

		case <-t.C:
			if err := l.Extend(ctx, key, token, opts.Lock); err != nil {
				return err
			}
		}
	}
}

func (l *Locker) TryLock(ctx context.Context, key, token string, ttl time.Duration) error {
	err := l.Client.SetArgs(ctx, key, token, redis.SetArgs{
		Mode: string(redis.NX),
		TTL:  ttl,
	}).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}

	return err
}

// LockWait waits until the lock is acquired.
func (l *Locker) LockWait(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	// NOTE: We don't use context for cancellation because it will be passed down.
	timeout := time.After(wait)
	tryLock := func() error {
		return l.TryLock(ctx, key, token, ttl)
	}

	var sleep time.Duration
	for {
		select {
		case <-timeout:
			err := tryLock()
			if errors.Is(err, ErrLocked) {
				return ErrLockWaitTimeout
			}

			return err
		case <-ctx.Done():
			return context.Cause(ctx)

		case <-time.After(sleep):
			err := tryLock()
			if errors.Is(err, ErrLocked) {
				sleep = rand.N(wait)

				continue
			}

			return err
		}
	}
}

func (l *Locker) Lock(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	if wait <= 0 {
		return l.TryLock(ctx, key, token, ttl)
	}

	return l.LockWait(ctx, key, token, ttl, wait)
}

// Unlocks the key with the given token.
func (l *Locker) Unlock(ctx context.Context, key, token string) error {
	n, err := l.Client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestString(token),
	}).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrExpired
	}
	return nil
}

func (l *Locker) Extend(ctx context.Context, key, token string, ttl time.Duration) error {
	val := []byte(token)
	err := l.Client.SetIFDEQ(ctx, key, val, helper.DigestString(token), ttl).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}
	return err
}

func newToken() string {
	return uuid.Must(uuid.NewV7()).String()
}
