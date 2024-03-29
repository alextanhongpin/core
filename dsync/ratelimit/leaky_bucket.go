package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const leakyBucket = "leaky_bucket"

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
	Now    time.Time
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
	keys := []string{fmt.Sprintf("ratelimit:%s:%s", leakyBucket, key)}
	args := []any{r.limit, r.period, r.burst, n, r.mockTime()}
	resp, err := r.client.FCall(ctx, leakyBucket, keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *LeakyBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *LeakyBucket) mockTime() int64 {
	if r.Now.IsZero() {
		return 0
	}

	return r.Now.UnixMilli()
}
