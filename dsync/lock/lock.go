package lock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrLocked indicates that the key is already locked.
	ErrLocked = errors.New("lock: already locked")

	ErrLockReleased = errors.New("lock: already released")

	ErrLockWaitTimeout = errors.New("lock: lock wait exceeded")

	ErrDone = errors.New("lock: done")
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

// Lock locks the given key until the function completes.
// If the lock cannot be acquired within the given wait, it will error.
// The lock is released after the function completes.
func (l *Locker) Lock(ctx context.Context, key string, ttl, wait time.Duration, fn func(ctx context.Context) error) (err error) {
	// Generate a random uuid as the lock value.
	val := uuid.New().String()
	if err := l.lockWait(ctx, key, val, ttl, wait); err != nil {
		return err
	}

	defer l.unlock(context.WithoutCancel(ctx), key, val)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(ErrDone)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := l.refresh(ctx, key, val, ttl)
		// Ignore if the fn is completed.
		if errors.Is(err, ErrDone) {
			return nil
		}

		return err
	})

	g.Go(func() error {
		// Signal the refresh to stop.
		defer cancel(ErrDone)

		return fn(ctx)
	})
	err = g.Wait()
	return
}

// lockWait attempts to acquire the lock. If the lock is already acquired, it
// will wait for the lock to be released.
// If the wait is less than or equal to 0, it will not wait.
func (l *Locker) lockWait(ctx context.Context, key, val string, ttl, wait time.Duration) error {
	noWait := wait <= 0
	if noWait {
		return l.lock(ctx, key, val, ttl)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, wait, ErrLockWaitTimeout)
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

func (l *Locker) refresh(ctx context.Context, key, val string, ttl time.Duration) error {
	t := time.NewTicker(ttl * 9 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := l.extend(ctx, key, val, ttl); err != nil {
				return err
			}
		}
	}
}

func (l *Locker) lock(ctx context.Context, key, val string, ttl time.Duration) error {
	ok, err := l.client.SetNX(ctx, key, val, ttl).Result()
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	if !ok {
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

	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}

	return nil
}

func (l *Locker) extend(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{val, ttl.Milliseconds()}
	err := extend.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLockReleased
	}

	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}

	return nil
}
