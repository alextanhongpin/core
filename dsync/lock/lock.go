package lock

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrLocked          = errors.New("lock: another process has acquired the lock")
	ErrConflict        = errors.New("lock: lock expired or acquired by another process")
	ErrLockWaitTimeout = errors.New("lock: failed to acquire lock within the wait duration")
)

type Options struct {
	LockTTL time.Duration // How long the lock is held.
	WaitTTL time.Duration // How long to wait for the lock to be acquired.
}

func (o *Options) Apply(opts ...Option) *Options {
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func NewOptions() *Options {
	return &Options{
		LockTTL: 10 * time.Second,
		WaitTTL: 1 * time.Minute,
	}
}

func (o *Options) Clone() *Options {
	return &Options{
		LockTTL: o.LockTTL,
		WaitTTL: o.WaitTTL,
	}
}

type Option func(o *Options)

func NoWait() Option {
	return func(o *Options) {
		o.WaitTTL = 0
	}
}

func WithLockTTL(t time.Duration) Option {
	return func(o *Options) {
		o.LockTTL = t
	}
}

func WithWaitTTL(t time.Duration) Option {
	return func(o *Options) {
		o.WaitTTL = t
	}
}

// Locker represents a distributed lock implementation using Redis.
type Locker struct {
	client *redis.Client
	opts   *Options
}

// New returns a pointer to Locker.
func New(client *redis.Client, opts *Options) *Locker {
	return &Locker{
		client: client,
		opts:   cmp.Or(opts, NewOptions()),
	}
}

// Do locks the given key until the function completes.
// If the lock cannot be acquired within the given wait, it will error.
// The lock is released after the function completes.
func (l *Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts ...Option) error {
	o := l.opts.Clone().Apply(opts...)

	// Generate a random uuid as the lock value.
	token, err := l.TryLock(ctx, key, o.LockTTL, o.WaitTTL)
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

	t := time.NewTicker(o.LockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case err := <-ch:
			return err
		case <-t.C:
			if err := l.Extend(ctx, key, token, o.LockTTL); err != nil {
				return err
			}
		}
	}
}

// TryLock attempts to acquire the lock. If the lock is already acquired, it
// will wait for the lock to be released.
// If the wait is less than or equal to 0, it will not wait.
func (l *Locker) TryLock(ctx context.Context, key string, ttl, wait time.Duration) (string, error) {
	noWait := wait <= 0
	if noWait {
		return l.Lock(ctx, key, ttl)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, wait, ErrLockWaitTimeout)
	defer cancel()

	var i int
	for {
		select {
		case <-ctx.Done():
			return "", context.Cause(ctx)
		default:
			token, err := l.Lock(ctx, key, ttl)
			if errors.Is(err, ErrLocked) {
				select {
				case <-ctx.Done():
					return "", context.Cause(ctx)
				case <-time.After(exponentialGrowthDecay(i)):
					i++
					continue
				}
			}

			return token, err
		}
	}
}

// Lock the key with the given ttl and returns a fencing token.
// If the lock is already acquired, it will return an error.
func (l *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := newFencingToken()
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

	return nil
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
	v, err := l.client.Do(ctx, "SET", key, string(value), "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}

	if err != nil {
		return "", false, err
	}

	return v.(string), true, nil
}

// combination of two curves. the duration increases exponentially in the beginning before beginning to decay.
// The idea is the wait duration should eventually be lesser and lesser over time.
func exponentialGrowthDecay(i int) time.Duration {
	x := float64(i)
	base := 1.0 + rand.Float64()
	switch {
	case x < 4: // intersection point rounded to 4
		base *= math.Pow(2, x)
	case x < 10:
		base *= 5 * math.Log(-0.9*x+10)
	default:
	}

	return time.Duration(base*100) * time.Millisecond
}

func newFencingToken() string {
	return uuid.Must(uuid.NewV7()).String()
}
