package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// GCRA implements the Generic Cell Rate Algorithm for smooth rate limiting.
// It provides better traffic shaping compared to fixed windows by avoiding
// burst behavior at window boundaries.
type GCRA struct {
	client *redis.Client
	burst  int
	limit  int
	period time.Duration
}

// NewGCRA creates a new GCRA rate limiter.
//
// Parameters:
//   - client: Redis client for distributed coordination
//   - limit: Maximum number of requests per period
//   - period: Time period for the rate limit
//   - burst: Additional burst capacity (0 = no burst allow)
//
// Example:
//
//	rl := NewGCRA(client, 100, time.Second, 10)  // 100 req/sec with 10 burst
func NewGCRA(client *redis.Client, limit int, period time.Duration, burst int) *GCRA {
	return &GCRA{
		limit:  limit,
		client: client,
		burst:  burst,
		period: period,
	}
}

// Allow checks if a single request is allow for the given key.
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error) {
	return g.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allow for the given key.
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := g.allowN(ctx, key, n)
	if err != nil {
		return false, err
	}

	return res.Allow, nil
}

func (g *GCRA) allowN(ctx context.Context, key string, n int) (*Result, error) {
	keys := []string{key}
	args := []any{
		g.burst,
		g.limit,
		g.period.Milliseconds(),
		n,
	}
	result, err := g.client.FCall(ctx, "gcra", keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}
	remaining := int(result[0])
	retryAfter := time.Duration(result[1]) * time.Millisecond

	return &Result{
		Allow:      remaining >= 0,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAfter: retryAfter,
	}, nil
}

// Limit performs a rate limit check and returns detailed information.
func (g *GCRA) Limit(ctx context.Context, key string) (*Result, error) {
	return g.LimitN(ctx, key, 1)
}

// LimitN performs a rate limit check for N requests and returns detailed information.
func (g *GCRA) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	return g.allowN(ctx, key, n)
}
