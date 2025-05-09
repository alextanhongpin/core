package lock

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const payload = "unlock"

var (
	ErrCanceled        = errors.New("lock: canceled")
	ErrConflict        = errors.New("lock: lock expired or acquired by another process")
	ErrLockTimeout     = errors.New("lock: exceeded lock duration")
	ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
	ErrLocked          = errors.New("lock: another process has acquired the lock")
)

// Locker represents a distributed lock implementation using Redis.
// Works on with a single redis node.
type Locker struct {
	client *redis.Client
}

// New returns a pointer to Locker.
func New(client *redis.Client) *Locker {
	return &Locker{
		client: client,
	}
}

func (l *Locker) Acquire(ctx context.Context, key string, lockTTL, waitTTL time.Duration) (func() error, error) {
	// Generate a random uuid as the lock value.
	token, err := l.TryLock(ctx, key, lockTTL, waitTTL)
	if err != nil {
		return nil, err
	}

	ch := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(ch)

		t := time.NewTicker(lockTTL * 7 / 10)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				ch <- context.Cause(ctx)
				return
			case <-t.C:
				if err := l.Extend(ctx, key, token, lockTTL); err != nil {
					ch <- err
					return
				}
			}
		}
	}()

	return func() error {
		cancel()
		// To ensure the unlock is called, we avoid using the same context.
		return errors.Join(<-ch, l.Unlock(context.WithoutCancel(ctx), key, token))
	}, nil
}

// DoTimeout ensures that the operation completes within the lockTTL.
// Use Do if you want the lock duration to be extended until completion.
func (l *Locker) DoTimeout(ctx context.Context, key string, fn func(ctx context.Context) error, lockTTL, waitTTL time.Duration) error {
	ctx, cancel := context.WithTimeoutCause(ctx, lockTTL, ErrLockTimeout)
	defer cancel()

	// Generate a random uuid as the lock value.
	_, err := l.TryLock(ctx, key, lockTTL, waitTTL)
	if err != nil {
		return err
	}
	ch := make(chan error, 1)
	go func() {
		ch <- fn(ctx)
	}()

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case err := <-ch:
		return err
	}
}

// Do locks the given key until the function completes.
// If the lock cannot be acquired within the given wait, it will error.
// The lock is released after the function completes.
// LockTTL: The duration the lock is held. Renewed every 7/10 of the LockTTL. Set it to at least 5s to ensure the lock has enough time to be renewed.
// WaitTTL: The duration to wait for the lock to be acquired. If set to 0, it will not wait and will return the error immediately.
func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, lockTTL, waitTTL time.Duration) error {
	// Generate a random uuid as the lock value.
	token, err := l.TryLock(ctx, key, lockTTL, waitTTL)
	if err != nil {
		return err
	}

	// To ensure the unlock is called, we avoid using the same context.
	defer l.Unlock(context.WithoutCancel(ctx), key, token)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a channel with a buffer of 1 to prevent goroutine leak.
	ch := make(chan error, 1)

	go func() {
		ch <- fn(ctx)
		close(ch)
	}()

	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case err := <-ch:
			return err
		case <-t.C:
			if err := l.Extend(ctx, key, token, lockTTL); err != nil {
				return err
			}
		}
	}
}

// TryLock attempts to acquire the lock. If the lock is already acquired, it
// will wait for the lock to be released.
// If the wait is less than or equal to 0, it will not wait.
func (l *Locker) TryLock(ctx context.Context, key string, ttl, wait time.Duration) (string, error) {
	nowait := wait <= 0
	if nowait {
		return l.Lock(ctx, key, ttl)
	}

	// Fire at the timeout moment before the wait duration.
	timeout := time.After(wait)

	pubsub := l.client.Subscribe(ctx, key)
	defer pubsub.Close()

	var i int
	for {
		sleep := exponentialBackoff(time.Second, time.Minute, i)

		// Sleep for the remaining time before the key expires.
		select {
		case msg := <-pubsub.Channel():
			if msg.Payload != payload {
				continue
			}

			token, err := l.Lock(ctx, key, ttl)
			if errors.Is(err, ErrLocked) {
				continue
			}

			return token, err
		case <-ctx.Done():
			return "", context.Cause(ctx)
		case <-timeout:
			token, err := l.Lock(ctx, key, ttl)
			if errors.Is(err, ErrLocked) {
				return "", ErrLockWaitTimeout
			}

			return token, err
		case <-time.After(sleep):
			token, err := l.Lock(ctx, key, ttl)
			if errors.Is(err, ErrLocked) {
				i++
				continue
			}

			return token, err
		}
	}
}

// Lock the key with the given ttl and returns a fencing token.
// If the lock is already acquired, it will return an error.
func (l *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := newToken()
	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", fmt.Errorf("lock: %w", err)
	}
	if !ok {
		return "", ErrLocked
	}

	return token, nil
}

// Unlocks the key with the given token.
func (l *Locker) Unlock(ctx context.Context, key, token string) error {
	keys := []string{key}
	argv := []any{token}
	err := unlock.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrConflict
	}

	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}

	return l.client.Publish(ctx, key, payload).Err()
}

func (l *Locker) Extend(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{val, ttl.Milliseconds()}
	err := extend.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrConflict
	}

	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}

	return nil
}

// Replace sets the value of the specified key to the provided new value, if
// the existing value matches the old value.
func (l *Locker) Replace(ctx context.Context, key, oldVal, newVal string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{oldVal, newVal, ttl.Milliseconds()}
	err := replace.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("replace: %w", ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	return nil
}

// LoadOrStore allows loading or storing a value to the key in a single
// operation.
// Returns true if the value is loaded, false if the value is stored.
func (l *Locker) LoadOrStore(ctx context.Context, key, value string, ttl time.Duration) (string, bool, error) {
	v, err := l.client.Do(ctx, "SET", key, value, "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}

	if err != nil {
		return "", false, err
	}

	return v.(string), true, nil
}

func newToken() string {
	return uuid.Must(uuid.NewV7()).String()
}

func exponentialBackoff(base, limit time.Duration, i int) time.Duration {
	return rand.N(min(base*time.Duration(math.Pow(2, float64(i))), limit))
}
