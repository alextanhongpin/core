package ratelimit

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// GCRA implements the Generic Cell Rate Algorithm: requests are spaced evenly
// at one every Period/Limit, and an idle key may additionally absorb Burst
// requests at once. Unlike FixedWindow it has no window boundary, so a client
// gets at most about Limit+Burst requests in any Period.
//
// Each key stores a single timestamp in Redis that expires once the key is
// fully replenished.
type GCRA struct {
	client redis.UniversalClient
	burst  int
	limit  int
	period time.Duration
}

var _ RateLimiter = (*GCRA)(nil)

// NewGCRA returns a GCRA limiter. Zero config fields take the defaults from
// DefaultConfig; invalid values return an error. Period/Limit must be at
// least one millisecond (for example 1000 per second is fine, 2000 per second
// is not).
//
// The client may be a *redis.Client, *redis.ClusterClient or any other
// redis.UniversalClient. It is not closed by the limiter. Run Setup once
// before using the limiter.
func NewGCRA(client redis.UniversalClient, cfg Config) (*GCRA, error) {
	if isNil(client) {
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

// MustNewGCRA is like NewGCRA but panics on error. Use it for package-level
// or startup configuration that is known to be valid.
func MustNewGCRA(client redis.UniversalClient, cfg Config) *GCRA {
	r, err := NewGCRA(client, cfg)
	if err != nil {
		panic(err)
	}
	return r
}

// Allow reports whether one request for key is allowed, consuming it if so.
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error) {
	return g.AllowN(ctx, key, 1)
}

// AllowN reports whether n requests for key are allowed, consuming all of
// them if so. A denied batch consumes nothing. A batch larger than Burst+1
// is never allowed.
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := g.LimitN(ctx, key, n)
	if err != nil {
		return false, err
	}
	return res.Allow, nil
}

// Limit is like Allow but returns the full Result, including how long to
// wait before retrying.
func (g *GCRA) Limit(ctx context.Context, key string) (*Result, error) {
	return g.LimitN(ctx, key, 1)
}

// LimitN is like AllowN but returns the full Result. n == 0 inspects the
// current state without consuming anything.
func (g *GCRA) LimitN(ctx context.Context, key string, n int) (*Result, error) {
	if err := validateRequest(key, n); err != nil {
		return nil, err
	}
	values, err := g.client.FCall(ctx, "rl_gcra", []string{key}, g.burst, g.limit, g.period.Milliseconds(), n).Int64Slice()
	if err != nil {
		return nil, wrapErr(err)
	}
	return parseResult(values)
}
