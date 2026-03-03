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

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrExpired         = errors.New("lock: lock expired")
	ErrLockTimeout     = errors.New("lock: exceeded lock duration")
	ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
	ErrLocked          = errors.New("lock: another process has acquired the lock")
)

type cacheable interface {
	CompareAndDelete(ctx context.Context, key string, old []byte) error
	CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error
	StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type backoff interface {
	Sleep() <-chan time.Time
}

type LockOption struct {
	//  The duration to wait for the lock to be available.
	Wait time.Duration
	// The duration for which the lock is held.
	Lock time.Duration
	// The ratio of the lock duration to refresh the lock.
	RefreshRatio float64
	// Optional token to identify the lock owner.
	Token string

	Backoff backoff
}

func NewLockOption() *LockOption {
	return &LockOption{
		Wait:         5 * time.Second,
		Lock:         30 * time.Second,
		RefreshRatio: 0.8,
		Backoff:      NewRandomBackoff(5 * time.Second),
	}
}

// Locker represents a distributed lock implementation using Redis.
// Works on with a single redis node.
type Locker struct {
	KeyMutex *KeyMutex
	Cache    cacheable
	Logger   *slog.Logger // Optional logger for debugging purposes.
}

// New returns a pointer to Locker.
func New(client *redis.Client) *Locker {
	return &Locker{
		KeyMutex: NewKeyMutex(),
		Cache:    cache.New(client),
		Logger:   slog.Default(), // Default logger, can be overridden.
	}
}

func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts *LockOption) error {
	unlock := l.KeyMutex.Lock(key)
	defer unlock()

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
	err := l.Cache.StoreOnce(ctx, key, []byte(token), ttl)
	if errors.Is(err, cache.ErrExists) {
		return ErrLocked
	}

	return nil
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
	err := l.Cache.CompareAndDelete(ctx, key, []byte(token))
	if err != nil {
		return fmt.Errorf("unlock: %w", mapError(err))
	}

	return nil
}

func (l *Locker) Extend(ctx context.Context, key, token string, ttl time.Duration) error {
	val := []byte(token)
	if err := l.Cache.CompareAndSwap(ctx, key, val, val, ttl); err != nil {
		return fmt.Errorf("extend: %w", mapError(err))
	}

	return nil
}

func newToken() string {
	return uuid.Must(uuid.NewV7()).String()
}

func NewRandomBackoff(base time.Duration) *RandomBackoff {
	return &RandomBackoff{
		base: base,
	}
}

type RandomBackoff struct {
	load bool
	base time.Duration
}

func (r *RandomBackoff) Sleep() <-chan time.Time {
	if !r.load {
		r.load = true
		return time.After(0)
	}

	return time.After(rand.N(r.base))
}

func NewExponentialBackoff(base, limit time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{
		base:  base,
		limit: limit,
	}
}

type ExponentialBackoff struct {
	attempts int
	base     time.Duration
	limit    time.Duration
}

func (b ExponentialBackoff) Sleep() <-chan time.Time {
	defer func() {
		b.attempts++
	}()

	sleep := rand.N(min(b.base*time.Duration(1<<b.attempts), b.limit))
	return time.After(sleep)
}

func mapError(err error) error {
	if errors.Is(err, cache.ErrNotExist) {
		return ErrExpired
	}

	if errors.Is(err, cache.ErrValueMismatch) {
		return ErrLocked
	}

	return err
}

// Example Prometheus integration
//
// import (
//   "github.com/prometheus/client_golang/prometheus"
//   "github.com/prometheus/client_golang/prometheus/promhttp"
//   "github.com/redis/go-redis/v9"
//   "github.com/alextanhongpin/core/dsync/lock"
//   "net/http"
// )
//
// func main() {
//   lockAttempts := prometheus.NewCounter(prometheus.CounterOpts{Name: "lock_attempts", Help: "Lock attempts."})
//   lockSuccess := prometheus.NewCounter(prometheus.CounterOpts{Name: "lock_success", Help: "Lock successes."})
//   lockFailures := prometheus.NewCounter(prometheus.CounterOpts{Name: "lock_failures", Help: "Lock failures."})
//   unlocks := prometheus.NewCounter(prometheus.CounterOpts{Name: "lock_unlocks", Help: "Unlocks."})
//   extends := prometheus.NewCounter(prometheus.CounterOpts{Name: "lock_extends", Help: "Extends."})
//   prometheus.MustRegister(lockAttempts, lockSuccess, lockFailures, unlocks, extends)
//
//   metrics := &lock.PrometheusLockMetrics{
//     LockAttempts: lockAttempts,
//     LockSuccess: lockSuccess,
//     LockFailures: lockFailures,
//     Unlocks: unlocks,
//     Extends: extends,
//   }
//   rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//   locker := lock.New(rdb, metrics)
//   http.Handle("/metrics", promhttp.Handler())
//   http.ListenAndServe(":8080", nil)
// }
