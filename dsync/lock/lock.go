package lock

import (
	"context"
	"errors"
	"fmt"
	"math"
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

	errDone = errors.New("lock: done")
)

type Options struct {
	LockTTL time.Duration
	WaitTTL time.Duration
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
	if opts == nil {
		opts = NewOptions()
	}

	return &Locker{
		client: client,
		opts:   opts,
	}
}

// Lock locks the given key until the function completes.
// If the lock cannot be acquired within the given wait, it will error.
// The lock is released after the function completes.
func (l *Locker) Lock(ctx context.Context, key string, fn func(ctx context.Context) error, opts ...Option) error {
	o := l.opts.Clone()
	for _, opt := range opts {
		opt(o)
	}

	// Generate a random uuid as the lock value.
	val := uuid.New().String()
	if err := l.lockWait(ctx, key, val, o.LockTTL, o.WaitTTL); err != nil {
		return err
	}

	defer l.unlock(context.WithoutCancel(ctx), key, val)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return l.refresh(ctx, key, val, o.LockTTL)
	})

	g.Go(func() error {
		if err := fn(ctx); err != nil {
			return err
		}
		return errDone
	})

	err := g.Wait()
	if errors.Is(err, errDone) {
		return nil
	}

	return err
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

	var i int
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
				case <-time.After(exponentialGrowthDecay(i)):
					i++
					continue
				}
			}

			return err
		}
	}
}

func (l *Locker) refresh(ctx context.Context, key, val string, ttl time.Duration) error {
	t := time.NewTicker(ttl * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
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
