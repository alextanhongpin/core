package ratelimit

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FixedWindow implements the Fixed Window algorithm.
type FixedWindow struct {
	client *redis.Client
	limit  int64
	period int64
}

type FixedWindowOption struct {
	Limit  int64
	Period time.Duration
}

func NewFixedWindow(client *redis.Client, opt *FixedWindowOption) *FixedWindow {
	fw := &FixedWindow{
		client: client,
		limit:  opt.Limit,
		period: opt.Period.Milliseconds(),
	}

	registerFunction(client)

	return fw
}

func (r *FixedWindow) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	keys := []string{key}
	args := []any{r.limit, r.period, n}
	resp, err := r.client.FCall(ctx, "fixed_window", keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *FixedWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}
