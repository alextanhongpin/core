package ratelimit

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FixedWindow allows up to Limit requests per key in each window of length
// Period. The window for a key starts at its first admitted request (it is
// not aligned to the wall clock) and the counter resets when it ends.
//
// It is simple and cheap (one Redis string per key) but lets a client send up
// to 2*Limit requests in a short time around a window boundary. Use GCRA when
// that matters.
type FixedWindow struct {
	client redis.UniversalClient
	limit  int
	period time.Duration
}

var _ RateLimiter = (*FixedWindow)(nil)

// NewFixedWindow returns a fixed-window limiter. Zero config fields take the
// defaults from DefaultConfig; invalid values return an error. Burst is
// ignored.
//
// The client may be a *redis.Client, *redis.ClusterClient or any other
// redis.UniversalClient. It is not closed by the limiter. Run Setup once
// before using the limiter.
func NewFixedWindow(client redis.UniversalClient, cfg Config) (*FixedWindow, error) {
	if isNil(client) {
		return nil, errors.New("ratelimit: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &FixedWindow{client: client, limit: cfg.Limit, period: cfg.Period}, nil
}

// MustNewFixedWindow is like NewFixedWindow but panics on error. Use it for
// package-level or startup configuration that is known to be valid.
func MustNewFixedWindow(client redis.UniversalClient, cfg Config) *FixedWindow {
	r, err := NewFixedWindow(client, cfg)
	if err != nil {
		panic(err)
	}
	return r
}

// Allow reports whether one request for key is allowed, consuming it if so.
func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return r.AllowN(ctx, key, 1)
}

// AllowN reports whether n requests for key are allowed, consuming all of
// them if so. A denied batch consumes nothing.
func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := r.LimitN(ctx, key, n)
	if err != nil {
		return false, err
	}
	return res.Allow, nil
}

// Limit is like Allow but returns the full Result, including how long to
// wait before retrying.
func (r *FixedWindow) Limit(ctx context.Context, key string) (*Result, error) {
	return r.LimitN(ctx, key, 1)
}

// LimitN is like AllowN but returns the full Result. n == 0 inspects the
// current state without consuming anything.
func (r *FixedWindow) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	if err := validateRequest(key, n); err != nil {
		return nil, err
	}
	values, err := r.client.FCall(ctx, "rl_fixed_window", []string{key}, r.limit, r.period.Milliseconds(), n).Int64Slice()
	if err != nil {
		return nil, wrapErr(err)
	}
	return parseResult(values)
}
