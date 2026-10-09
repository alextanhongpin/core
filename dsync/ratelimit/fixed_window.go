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
	period time.Duration
}

// NewFixedWindow defaults and validates a configuration value; client remains borrowed.
func NewFixedWindow(client *redis.Client, cfg Config) (*FixedWindow, error) {
	if client == nil {
		return nil, errors.New("ratelimit: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &FixedWindow{client: client, limit: cfg.Limit, period: cfg.Period}, nil
}
func MustNewFixedWindow(client *redis.Client, cfg Config) *FixedWindow {
	r, err := NewFixedWindow(client, cfg)
	if err != nil {
		panic(err)
	}
	return r
}

// AllowN checks if N requests are allowed for the given key.
func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := r.LimitN(ctx, key, n)
	if err != nil {
		return false, err
	}
	return res.Allow, nil
}

// Allow checks if a single request is allowed for the given key.
func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	if err := validateRequest(key, n); err != nil {
		return nil, err
	}
	values, err := r.client.FCall(ctx, "rl_fixed_window", []string{key}, r.limit, r.period.Milliseconds(), n).Int64Slice()
	if err != nil {
		return nil, err
	}
	return parseResult(values)
}

func (r *FixedWindow) Limit(ctx context.Context, key string) (*Result, error) {
	return r.LimitN(ctx, key, 1)
}
