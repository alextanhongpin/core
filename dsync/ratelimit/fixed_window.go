package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const fixedWindow = "fixed_window"

// FixedWindow implements the Fixed Window algorithm.
type FixedWindow struct {
	client *redis.Client
	limit  int64
	period int64
	Now    time.Time // Immutable time for testing.
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
	keys := []string{fmt.Sprintf("ratelimit:%s:%s", fixedWindow, key)}
	args := []any{r.limit, r.period, n, r.mockTime()}
	resp, err := r.client.FCall(ctx, fixedWindow, keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *FixedWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) mockTime() int64 {
	if r.Now.IsZero() {
		return 0
	}

	return r.Now.UnixMilli()
}
