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
	"math"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	redis "github.com/redis/go-redis/v9"
)

const payload = "unlock"

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

type backOffPolicy interface {
	BackOff(i int) time.Duration
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
}

func NewLockOption() *LockOption {
	return &LockOption{
		Wait:         5 * time.Second,
		Lock:         30 * time.Second,
		RefreshRatio: 0.8,
	}
}

// MetricsCollector defines the interface for collecting lock metrics.
type MetricsCollector interface {
	IncLockAttempts()
	IncLockSuccess()
	IncLockFailures()
	IncUnlocks()
	IncExtends()
}

type AtomicLockMetrics struct {
	lockAttempts int64
	lockSuccess  int64
	lockFailures int64
	unlocks      int64
	extends      int64
}

func (m *AtomicLockMetrics) IncLockAttempts() { atomic.AddInt64(&m.lockAttempts, 1) }
func (m *AtomicLockMetrics) IncLockSuccess()  { atomic.AddInt64(&m.lockSuccess, 1) }
func (m *AtomicLockMetrics) IncLockFailures() { atomic.AddInt64(&m.lockFailures, 1) }
func (m *AtomicLockMetrics) IncUnlocks()      { atomic.AddInt64(&m.unlocks, 1) }
func (m *AtomicLockMetrics) IncExtends()      { atomic.AddInt64(&m.extends, 1) }

// PrometheusLockMetrics implements MetricsCollector using prometheus metrics.
type PrometheusLockMetrics struct {
	LockAttempts prometheus.Counter
	LockSuccess  prometheus.Counter
	LockFailures prometheus.Counter
	Unlocks      prometheus.Counter
	Extends      prometheus.Counter
}

func (m *PrometheusLockMetrics) IncLockAttempts() { m.LockAttempts.Inc() }
func (m *PrometheusLockMetrics) IncLockSuccess()  { m.LockSuccess.Inc() }
func (m *PrometheusLockMetrics) IncLockFailures() { m.LockFailures.Inc() }
func (m *PrometheusLockMetrics) IncUnlocks()      { m.Unlocks.Inc() }
func (m *PrometheusLockMetrics) IncExtends()      { m.Extends.Inc() }

// Locker represents a distributed lock implementation using Redis.
// Works on with a single redis node.
type Locker struct {
	mu *KeyedMutex

	BackOff          backOffPolicy // Optional backoff policy for retrying lock acquisition.
	Cache            cacheable
	Logger           *slog.Logger // Optional logger for debugging purposes.
	metricsCollector MetricsCollector
}

// New returns a pointer to Locker.
func New(client *redis.Client, collectors ...MetricsCollector) *Locker {
	var collector MetricsCollector
	if len(collectors) > 0 && collectors[0] != nil {
		collector = collectors[0]
	} else {
		collector = &AtomicLockMetrics{}
	}
	return &Locker{
		mu: NewKeyedMutex(),
		BackOff: &exponentialBackOff{
			base:  time.Second, // Base backoff duration.
			limit: time.Minute, // Maximum backoff duration.
		},
		Cache:            cache.New(client),
		Logger:           slog.Default(), // Default logger, can be overridden.
		metricsCollector: collector,
	}
}

func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts *LockOption) error {
	mu := l.mu.Key(key)
	mu.Lock()
	defer mu.Unlock()

	if opts == nil {
		opts = NewLockOption()
	}
	token := cmp.Or(opts.Token, newToken())
	if opts.Lock <= 0 {
		return fmt.Errorf("lock duration must be greater than zero, got %v", opts.Lock)
	}

	var cancel context.CancelFunc

	noRefresh := opts.RefreshRatio <= 0
	if noRefresh {
		ctx, cancel = context.WithTimeoutCause(ctx, opts.Lock, ErrLockTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if err := l.Lock(ctx, key, token, opts.Lock, opts.Wait); err != nil {
		return err
	}
	defer func() {
		if unlockErr := l.Unlock(context.WithoutCancel(ctx), key, token); unlockErr != nil {
			l.Logger.Error("failed to unlock", "key", key, "err", unlockErr)
		}
	}()

	if noRefresh {
		return fn(ctx)
	}

	// Create a channel with a buffer of 1 to prevent goroutine leak.
	ch := make(chan error, 1)

	go func() {
		ch <- fn(ctx)
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

func (l *Locker) Lock(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	if wait <= 0 {
		return l.TryLock(ctx, key, token, ttl)
	}

	// Fire at the timeout moment before the wait duration.
	timeout := time.After(wait)

	var i int
	for {
		sleep := l.BackOff.BackOff(i)

		// Sleep for the remaining time before the key expires.
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-timeout:
			err := l.TryLock(ctx, key, token, ttl)
			if errors.Is(err, ErrLocked) {
				return ErrLockWaitTimeout
			}

			return err
		case <-time.After(sleep):
			err := l.TryLock(ctx, key, token, ttl)
			if errors.Is(err, ErrLocked) {
				i++
				continue
			}

			return err
		}
	}
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

type exponentialBackOff struct {
	base  time.Duration
	limit time.Duration
}

func (b exponentialBackOff) BackOff(i int) time.Duration {
	return rand.N(min(b.base*time.Duration(math.Pow(2, float64(i))), b.limit))
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
