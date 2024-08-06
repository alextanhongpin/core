package singleflight

import (
	"context"
	"encoding/json"
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
	ErrTokenMismatch   = errors.New("singleflight: token mismatch")
	ErrLockWaitTimeout = errors.New("singleflight: lock wait timeout")
)

type Group[T any] struct {
	client  *redis.Client
	KeepTTL time.Duration
	LockTTL time.Duration
	WaitTTL time.Duration
}

func New[T any](client *redis.Client) *Group[T] {
	return &Group[T]{
		client:  client,
		KeepTTL: 1 * time.Hour,
		LockTTL: 10 * time.Second,
		WaitTTL: 1 * time.Minute,
	}
}

func (g *Group[T]) Do(ctx context.Context, key string, fn func(context.Context) (T, error)) (v T, err error, shared bool) {
	token := []byte(uuid.New().String())
	b, loaded, err := g.loadOrStore(ctx, key, token, g.LockTTL)
	if err != nil {
		return v, err, false
	}

	if loaded {
		if !isUUID(b) {
			err = json.Unmarshal(b, &v)
			if err != nil {
				return v, err, false
			}
			return v, nil, true
		}

		v, err = g.wait(ctx, key, g.WaitTTL)
		if err != nil {
			// If all attempts failed, just fail fast without retrying.
			// Let another process retry.
			return v, err, false
		}

		return v, nil, true
	}

	// Use a separate context to avoid cancellation.
	defer g.unlock(context.WithoutCancel(ctx), key, token)

	ch := make(chan T)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return g.refresh(ctx, ch, key, token, g.LockTTL, g.KeepTTL)
	})

	eg.Go(func() error {
		v, err = fn(ctx)
		if err != nil {
			return err
		}

		ch <- v
		close(ch)

		return nil
	})

	if err := eg.Wait(); err != nil {
		return v, err, false
	}

	return v, nil, false
}

func (g *Group[T]) wait(ctx context.Context, key string, ttl time.Duration) (v T, err error) {
	ctx, cancel := context.WithTimeoutCause(ctx, ttl, ErrLockWaitTimeout)
	defer cancel()

	var i int
	for {
		select {
		case <-ctx.Done():
			return v, context.Cause(ctx)
		// Retry at most 10 times.
		case <-time.After(exponentialGrowthDecay(i)):
			i++
			b, err := g.client.Get(ctx, key).Bytes()
			if err != nil {
				return v, fmt.Errorf("wait: %w", err)
			}

			// Is pending.
			if isUUID(b) {
				continue
			}

			err = json.Unmarshal(b, &v)
			return v, err
		}
	}
}

func (g *Group[T]) refresh(ctx context.Context, ch chan T, key string, value []byte, lockTTL, keepTTL time.Duration) error {
	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case v := <-ch:
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}

			return g.replace(ctx, key, value, b, keepTTL)
		case <-t.C:
			if err := g.extend(ctx, key, value, lockTTL); err != nil {
				return err
			}
		}
	}
}

func (g *Group[T]) lock(ctx context.Context, key string, value []byte, ttl time.Duration) (locked bool, err error) {
	return g.client.SetNX(ctx, key, value, ttl).Result()
}

// unlock releases the lock on the specified key using the provided value.
func (g *Group[T]) unlock(ctx context.Context, key string, value []byte) error {
	keys := []string{key}
	argv := []any{value}
	err := unlock.Run(ctx, g.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("unlock: %w", ErrTokenMismatch)
	}
	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}

	return nil
}

func (g *Group[T]) extend(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{val, ttl.Milliseconds()}
	err := extend.Run(ctx, g.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("extend: %w", ErrTokenMismatch)
	}
	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}

	return nil
}

// replace sets the value of the specified key to the provided new value, if
// the existing value matches the old value.
func (g *Group[T]) replace(ctx context.Context, key string, oldVal, newVal []byte, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{oldVal, newVal, ttl.Milliseconds()}
	err := replace.Run(ctx, g.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("replace: %w", ErrTokenMismatch)
	}
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	return nil
}

func (g *Group[T]) loadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, bool, error) {
	v, err := g.client.Do(ctx, "SET", key, string(value), "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return []byte(v.(string)), true, nil
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
}

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
