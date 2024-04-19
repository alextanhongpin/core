package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FixedWindow implements the Fixed Window algorithm.
type FixedWindow struct {
	client *redis.Client
	opt    *FixedWindowOption
	Now    func() time.Time
}

type FixedWindowOption struct {
	Limit  int64
	Period time.Duration
}

func NewFixedWindow(client *redis.Client, opt *FixedWindowOption) *FixedWindow {
	if opt == nil {
		panic("ratelimit: option is nil")
	}
	if opt.Limit <= 0 {
		panic("ratelimit: limit is invalid")
	}
	if opt.Period <= 0 {
		panic("ratelimit: period is invalid")
	}
	return &FixedWindow{
		client: client,
		opt:    opt,
		Now:    time.Now,
	}
}

func (r *FixedWindow) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	key = r.buildKey(r.Now(), key)
	now := r.Now()
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if n == 1 {
		err = r.client.PExpire(ctx, key, r.opt.Period).Err()
		if err != nil {
			return nil, err
		}
	}
	start := now.Truncate(r.opt.Period)
	end := start.Add(r.opt.Period)

	retryAt := now
	if n < r.opt.Limit {
		retryAt = end
	}

	return &Result{
		Allow:     n <= r.opt.Limit,
		Remaining: max(0, r.opt.Limit-n),
		Limit:     r.opt.Limit,
		ResetAt:   end,
		RetryAt:   retryAt,
	}, nil
}

func (r *FixedWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) buildKey(now time.Time, key string) string {
	t := now.Truncate(r.opt.Period).Format(time.RFC3339Nano)
	// Set the key first to allow users to search their key by prefix.
	// The ratelimit:fixed_window is used to identify the
	// algorithm used, in case users switched the
	// implementation.
	return fmt.Sprintf("%s:ratelimit:fixed_window:%s", key, t)
}
