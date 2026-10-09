package ratelimit

import (
	"context"
	_ "embed"
	"errors"
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

// NewGCRA defaults and validates a configuration value; client remains borrowed.
func NewGCRA(client *redis.Client, cfg Config) (*GCRA, error) {
	if client == nil {
		return nil, errors.New("ratelimit: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.validateGCRA(); err != nil {
		return nil, err
	}
	return &GCRA{client: client, limit: cfg.Limit, period: cfg.Period, burst: cfg.Burst}, nil
}
func MustNewGCRA(client *redis.Client, cfg Config) *GCRA {
	r, err := NewGCRA(client, cfg)
	if err != nil {
		panic(err)
	}
	return r
}

// Allow checks if a single request is allow for the given key.
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error) {
	return g.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allow for the given key.
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := g.LimitN(ctx, key, n)
	if err != nil {
		return false, err
	}

	return res.Allow, nil
}

// LimitN performs a rate limit check for N requests and returns detailed information.
func (g *GCRA) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	if err := validateRequest(key, n); err != nil {
		return nil, err
	}
	values, err := g.client.FCall(ctx, "rl_gcra", []string{key}, g.burst, g.limit, g.period.Milliseconds(), n).Int64Slice()
	if err != nil {
		return nil, err
	}
	return parseResult(values)
}

// Limit performs a rate limit check and returns detailed information.
func (g *GCRA) Limit(ctx context.Context, key string) (*Result, error) {
	return g.LimitN(ctx, key, 1)
}
