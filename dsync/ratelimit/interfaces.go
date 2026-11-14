// Package ratelimit provides distributed rate limiters backed by Redis, so
// that several instances of a service share one quota per key (per user,
// per API key, per IP, ...).
//
// Two algorithms are available:
//
//   - FixedWindow: at most Limit requests per Period, counted in windows that
//     start at a key's first request. Simple, but allows bursts of up to
//     2*Limit around a window boundary.
//   - GCRA: requests are spread evenly (one every Period/Limit) with an
//     optional Burst allowance. Smoother, with no boundary bursts.
//
// Each decision is made atomically by a Lua function inside Redis using the
// Redis server clock, so client clock skew does not matter. Install the
// functions once at startup with Setup.
//
// Minimal usage:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	if err := ratelimit.Setup(ctx, client); err != nil {
//		return err
//	}
//	limiter, err := ratelimit.NewGCRA(client, ratelimit.Config{Limit: 10, Period: time.Second})
//	if err != nil {
//		return err
//	}
//	res, err := limiter.Limit(ctx, "user:123")
//	if err != nil {
//		return err // Redis unavailable or invalid input; not a denial.
//	}
//	if !res.Allow {
//		// Reject, e.g. HTTP 429 with Retry-After: res.RetryAfter.
//	}
package ratelimit

import (
	"context"
	"errors"
	"time"
)

// RateLimiter is implemented by *FixedWindow and *GCRA. Use it to swap
// algorithms or to fake the limiter in tests.
type RateLimiter interface {
	// Allow reports whether a single request for key is permitted.
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN reports whether n requests for key are permitted, all or nothing.
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// Limit returns a detailed Result for a single request.
	Limit(ctx context.Context, key string) (*Result, error)

	// LimitN returns a detailed Result for n requests.
	LimitN(ctx context.Context, key string, n int) (*Result, error)
}

// Result describes a rate limit decision. All durations are computed by
// Redis with millisecond precision.
type Result struct {
	// Allow is true if the requests were admitted (and consumed).
	Allow bool

	// Remaining is how many more requests could be admitted right now,
	// after this decision.
	Remaining int

	// ResetAfter is how long until the key is back to full capacity
	// (FixedWindow: until the current window ends).
	ResetAfter time.Duration

	// RetryAfter is how long to wait before the same request can succeed.
	// It is zero when Allow is true. For GCRA it is also zero when the batch
	// is larger than Burst+1 and can never succeed; FixedWindow then reports
	// the end of the current window.
	RetryAfter time.Duration
}

func parseResult(values []int64) (*Result, error) {
	if len(values) != 4 {
		return nil, errors.New("ratelimit: invalid Redis response")
	}
	return &Result{
		Allow:      values[0] == 1,
		Remaining:  int(values[1]),
		RetryAfter: time.Duration(values[2]) * time.Millisecond,
		ResetAfter: time.Duration(values[3]) * time.Millisecond,
	}, nil
}
