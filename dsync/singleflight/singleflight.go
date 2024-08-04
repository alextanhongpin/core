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
	ErrTokenMismatch = errors.New("singleflight: token mismatch")
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
	ok, err := g.lock(ctx, key, token, g.LockTTL)
	if err != nil {
		return v, err, false
	}

	// The data already exists.
	if !ok {
		v, err = g.wait(ctx, key, g.WaitTTL)
		if err != nil {
			return v, err, false
		}

		return v, nil, true
	}

	// Use a separate context to avoid cancellation.
	defer g.unlock(context.Background(), key, token)

	grp, gctx := errgroup.WithContext(ctx)
	gctx, cancel := context.WithCancel(gctx)
	defer cancel()

	grp.Go(func() error {
		return g.refresh(gctx, key, token, g.LockTTL)
	})

	grp.Go(func() error {
		// Cancel the refresh after loading and replacing
		// completes.
		defer cancel()

		v, err = g.load(gctx, key, token, fn)
		return err
	})

	if err := grp.Wait(); err != nil {
		return v, err, false
	}

	return v, nil, false
}

func (g *Group[T]) load(ctx context.Context, key string, token []byte, fn func(ctx context.Context) (T, error)) (v T, err error) {
	v, err = fn(ctx)
	if err != nil {
		return v, err
	}

	b, err := json.Marshal(v)
	if err != nil {
		return v, err
	}

	if err := g.replace(ctx, key, []byte(token), b, g.KeepTTL); err != nil {
		return v, err
	}

	return v, nil
}

func (g *Group[T]) wait(ctx context.Context, key string, ttl time.Duration) (v T, err error) {
	ctx, cancel := context.WithTimeout(ctx, ttl)
	defer cancel()

	var i int
	for {
		time.Sleep(exponentialBackoff(i))
		i++

		b, err := g.client.Get(ctx, key).Bytes()
		if err != nil {
			return v, fmt.Errorf("%w: wait", err)
		}
		// Still not completed.
		if isUUID(b) {
			continue
		}

		err = json.Unmarshal(b, &v)
		return v, err
	}
}

func (g *Group[T]) refresh(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	t := time.NewTicker(ttl * 9 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := g.extend(ctx, key, val, ttl); err != nil {
				return err
			}
		}
	}
}

func (g *Group[T]) lock(ctx context.Context, key string, value []byte, ttl time.Duration) (locked bool, err error) {
	return g.client.SetNX(ctx, key, value, ttl).Result()
}

// unlock releases the lock on the specified key using the provided value.
func (g *Group[T]) unlock(ctx context.Context, key string, val []byte) error {
	keys := []string{key}
	argv := []any{val}
	err := unlock.Run(ctx, g.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: unlock", ErrTokenMismatch)
	}
	if err != nil {
		return fmt.Errorf("%w: unlock", err)
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
		return fmt.Errorf("%w: extend", err)
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
		return fmt.Errorf("%w: replace", ErrTokenMismatch)
	}
	if err != nil {
		return fmt.Errorf("%w: replace", err)
	}

	return nil
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
}

func exponentialBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}
	base := float64(25 * time.Millisecond)
	cap := float64(time.Minute)
	jitter := 1.0 + rand.Float64()
	return time.Duration(min(cap, jitter*base*math.Pow(2, float64(attempts))))
}
