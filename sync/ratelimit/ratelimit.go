package ratelimit

import "time"

type Result struct {
	Allow      bool
	Limit      int64
	Remaining  int64
	RetryAfter time.Duration
}

// Throttler limits the number of requests per second.
type Throttler interface {
	Allow() bool
	AllowN(int) bool
}

// Regulator ensures a constant flow of request.
type Regulator interface {
	Allow() bool
	AllowN(int) bool
}

type RateLimiter struct {
	throttler Throttler
	regulator Regulator
}

func New(
	throttler Throttler,
	regulator Regulator,
) *RateLimiter {
	return &RateLimiter{
		throttler: throttler,
		regulator: regulator,
	}
}

func (r *RateLimiter) Allow() bool {
	return r.throttler.Allow() && r.regulator.Allow()
}

func (r *RateLimiter) AllowN(n int) bool {
	return r.throttler.AllowN(n) && r.regulator.AllowN(n)
}
