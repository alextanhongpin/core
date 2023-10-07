package ratelimit

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type LeakyBucketOption struct {
	Limit  int64
	Period time.Duration
	Burst  int64
}

// LeakyBucket implements the Genetic Cell Rate algorithm.
type LeakyBucket struct {
	client *redis.Client
	limit  int64
	period int64
	burst  int64
}

func NewLeakyBucket(client *redis.Client, opt *LeakyBucketOption) *LeakyBucket {
	lb := &LeakyBucket{
		client: client,
		limit:  opt.Limit,
		period: opt.Period.Milliseconds(),
		burst:  opt.Burst,
	}

	registerFunction(client)

	return lb
}

func (r *LeakyBucket) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	keys := []string{key}
	args := []any{r.limit, r.period, r.burst, n}
	resp, err := r.client.FCall(ctx, "leaky_bucket", keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *LeakyBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}
