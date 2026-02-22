package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FixedWindow implements the Fixed Window algorithm for rate limiting.
// It divides time into fixed intervals and allows a specified number
// of requests per interval.
type FixedWindow struct {
	client *redis.Client
	limit  int
	period time.Duration
}

// NewFixedWindow creates a new Fixed Window rate limiter.
//
// Parameters:
//   - client: Redis client for distributed coordination
//   - limit: Maximum number of requests per window
//   - period: Duration of each window
//
// Example:
//
//	rl := NewFixedWindow(client, 1000, time.Hour)  // 1000 requests per hour
func NewFixedWindow(client *redis.Client, limit int, period time.Duration) *FixedWindow {
	return &FixedWindow{
		client: client,
		limit:  limit,
		period: period,
	}
}

// allowN checks if N requests are allowed for the given key.
func (r *FixedWindow) allowN(ctx context.Context, key string, n int) (int, error) {
	keys := []string{key}
	args := []any{
		r.limit,
		r.period.Milliseconds(),
		n,
	}
	remaining, err := r.client.FCall(ctx, "fixed_window", keys, args...).Int()
	return remaining, err
}

// AllowN checks if N requests are allowed for the given key.
func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	remaining, err := r.allowN(ctx, key, n)
	if err != nil {
		return false, err
	}
	return remaining >= 0, nil
}

// Allow checks if a single request is allowed for the given key.
func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	remaining, err := r.allowN(ctx, key, n)
	if err != nil {
		return nil, err
	}

	resetAfter, err := r.client.PTTL(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	result := &Result{
		Allow:      remaining >= 0,
		Remaining:  remaining,
		ResetAfter: resetAfter,
		RetryAfter: resetAfter,
	}
	if remaining > 0 {
		// Can immediately retry.
		result.RetryAfter = 0
	}
	return result, nil
}

func (r *FixedWindow) Limit(ctx context.Context, key string) (*Result, error) {
	return r.LimitN(ctx, key, 1)
}
