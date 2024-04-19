package lock

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	// ErrLocked indicates that the key is already locked.
	ErrLocked = errors.New("lock: already locked")

	ErrLockReleased = errors.New("lock: already released")

	ErrLockWaitTimeout = errors.New("lock: wait timeout exceeded")
)

// Locker represents a distributed lock implementation using Redis.
type Locker struct {
	client *redis.Client
}

// New returns a pointer to Locker.
func New(client *redis.Client) *Locker {
	return &Locker{
		client: client,
	}
}

// lockWait attempts to acquire the lock. If the lock is already acquired, it
// will wait for the lock to be released.
// If the timeout is less than or equal to 0, it will not wait.
func (l *Locker) lockWait(ctx context.Context, key, val string, ttl, timeout time.Duration) error {
	// No wait
	if timeout <= 0 {
		return l.lock(ctx, key, val, ttl)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, timeout, ErrLockWaitTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
			err := l.lock(ctx, key, val, ttl)
			if errors.Is(err, ErrLocked) {
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
					// Retry at most 10 times.
				case <-time.After(time.Duration(rand.Intn(int(ttl)/10)) + ttl/10):
					continue
				}
			}

			return err
		}
	}
}

func (l *Locker) refresh(ctx context.Context, key, val string, ttl time.Duration) chan error {
	ch := make(chan error)

	go func() {
		defer close(ch)

		t := time.NewTicker(ttl / 9)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				ch <- context.Cause(ctx)
				return
			case <-t.C:
				if err := l.extend(ctx, key, val, ttl); err != nil {
					ch <- err
					return
				}
			}
		}
	}()

	return ch
}

func (l *Locker) Lock(ctx context.Context, key string, ttl, timeout time.Duration, fn func(ctx context.Context) error) (err error) {
	// Generate a random uuid as the lock value.
	val := uuid.New().String()
	if err := l.lockWait(ctx, key, val, ttl, timeout); err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, l.unlock(context.WithoutCancel(ctx), key, val))
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := l.refresh(ctx, key, val, ttl)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer cancel()
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case refreshErr := <-ch:
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}

				err = errors.Join(err, refreshErr)
				return
			}
		}
	}()

	err = fn(ctx)
	cancel()
	wg.Wait()
	return
}

func (l *Locker) lock(ctx context.Context, key, val string, ttl time.Duration) error {
	ok, err := l.client.SetNX(ctx, key, val, ttl).Result()
	if !ok || errors.Is(err, redis.Nil) {
		return ErrLocked
	}

	return nil
}

func (l *Locker) unlock(ctx context.Context, key, val string) error {
	keys := []string{key}
	argv := []any{val}
	err := unlock.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLockReleased
	}

	return err
}

func (l *Locker) extend(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{val, formatMs(ttl)}
	err := extend.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLockReleased
	}

	return err
}

// copied from redis source code
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}
