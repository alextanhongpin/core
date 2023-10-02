package ratelimit

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type GCRAOption struct {
	Limit  int64
	Period time.Duration
	Burst  int64
}

// GCRA implements the Genetic Cell Rate algorithm.
type GCRA struct {
	client *redis.Client
	limit  int64
	period int64
	burst  int64
}

func NewGCRA(client *redis.Client, opt *GCRAOption) *GCRA {
	gcra := &GCRA{
		client: client,
		limit:  opt.Limit,
		period: opt.Period.Milliseconds(),
		burst:  opt.Burst,
	}

	registerFunction(client)

	return gcra
}

func (r *GCRA) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	keys := []string{key}
	args := []any{r.limit, r.period, r.burst, n}
	resp, err := r.client.FCall(ctx, "gcra", keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *GCRA) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}
