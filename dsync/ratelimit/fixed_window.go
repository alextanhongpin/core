package ratelimit

import (
	"context"
	_ "embed"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed fixed_window.lua
var fixedWindowScript string

var fixedWindow = redis.NewScript(fixedWindowScript)

// FixedWindow implements the Fixed Window algorithm.
type FixedWindow struct {
	client *redis.Client
	limit  int
	period int64
}

func NewFixedWindow(client *redis.Client, limit int, period time.Duration) *FixedWindow {
	return &FixedWindow{
		client: client,
		limit:  limit,
		period: period.Milliseconds(),
	}
}

func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	keys := []string{key}
	argv := []any{
		r.limit,
		r.period,
		n,
	}
	ok, err := fixedWindow.Run(ctx, r.client, keys, argv...).Int()
	if err != nil {
		return false, err
	}
	return ok == 1, nil
}

func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) Remaining(ctx context.Context, key string) (int, error) {
	n, err := r.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return r.limit, nil
	}
	if err != nil {
		return 0, err
	}

	return r.limit - n, nil
}

func (r *FixedWindow) ResetAfter(ctx context.Context, key string) (time.Duration, error) {
	remaining, err := r.Remaining(ctx, key)
	if err != nil {
		return 0, err
	}
	if remaining > 0 {
		return 0, nil
	}

	d, err := r.client.PTTL(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return d, err
}
