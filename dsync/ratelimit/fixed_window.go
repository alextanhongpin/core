package ratelimit

import (
	"context"
	_ "embed"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FixedWindow implements the Fixed Window algorithm for rate limiting.
// It divides time into fixed intervals and allows a specified number
// of requests per interval.
type FixedWindow struct {
	client *redis.Client
	limit  int
	period int64
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
		period: period.Milliseconds(),
	}
}

// AllowN checks if N requests are allowed for the given key.
func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	keys := []string{key}
	args := []any{
		r.limit,
		r.period,
		n,
	}
	ok, err := r.client.FCall(ctx, "fixed_window", keys, args...).Int()
	if err != nil {
		return false, err
	}
	return ok == 1, nil
}

// Allow checks if a single request is allowed for the given key.
func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

// Remaining returns the number of requests remaining in the current window.
func (r *FixedWindow) Remaining(ctx context.Context, key string) (int, error) {
	n, err := r.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return r.limit, nil
	}
	if err != nil {
		return 0, err
	}

	remaining := max(r.limit-n, 0)
	return remaining, nil
}

// ResetAfter returns the duration until the current window resets.
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

// Check performs a rate limit check and returns detailed information.
func (r *FixedWindow) Check(ctx context.Context, key string) (*Result, error) {
	return r.CheckN(ctx, key, 1)
}

// CheckN performs a rate limit check for N requests and returns detailed information.
func (r *FixedWindow) CheckN(ctx context.Context, key string, n int) (*Result, error) {
	allowed, err := r.AllowN(ctx, key, n)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Allowed: allowed,
	}

	// Get remaining count
	remaining, err := r.Remaining(ctx, key)
	if err != nil {
		return nil, err
	}
	result.Remaining = remaining

	// Get reset time
	resetAfter, err := r.ResetAfter(ctx, key)
	if err != nil {
		return nil, err
	}
	result.ResetAfter = resetAfter

	// If not allowed, suggest retry after reset
	if !allowed {
		result.RetryAfter = resetAfter
	}

	return result, nil
}
