// Package ratelimit provides distributed rate limiting algorithms using Redis.
//
// This package implements multiple rate limiting algorithms suitable for different
// use cases and traffic patterns:
//
//   - Fixed Window: Simple burst-tolerant rate limiting with fixed time windows
//   - GCRA: Smooth rate limiting using Generic Cell Rate Algorithm
//
// All implementations are distributed and coordinated through Redis, making them
// suitable for multi-instance applications.
//
// Example usage:
//
//	// Fixed Window: 1000 requests per hour
//	fw := ratelimit.NewFixedWindow(redisClient, 1000, time.Hour)
//	allowed, err := fw.Allow(ctx, "user:123")
//
//	// GCRA: 100 requests per second with 10 burst capacity
//	gcra := ratelimit.NewGCRA(redisClient, 100, time.Second, 10)
//	allowed, err := gcra.Allow(ctx, "api:key")
//
// Performance characteristics:
//   - Fixed Window: Simple, burst-friendly, predictable resets
//   - GCRA: Smooth, burst-configurable, better traffic shaping
//
// Both algorithms are atomic and race-condition free thanks to Redis Lua scripts.
package ratelimit

import (
	"context"
	"time"
)

// RateLimiter defines the common interface for all rate limiting algorithms.
type RateLimiter interface {
    // Allow returns true if a single request for `key` is permitted.
    Allow(ctx context.Context, key string) (bool, error)

    // AllowN returns true if `n` requests for `key` are permitted.
    AllowN(ctx context.Context, key string, n int) (bool, error)

    // Limit returns a detailed Result for a single request.
    Limit(ctx context.Context, key string) (*Result, error)

    // LimitN returns a detailed Result for `n` requests.
    LimitN(ctx context.Context, key string, n int) (*Result, error)
}

// Result contains detailed information about a rate limit check.
type Result struct {
	// Allow indicates if the request was allowed.
	Allow bool

	// Remaining is the number of requests remaining (if applicable).
	Remaining int

	// ResetAfter is the duration until the limit resets (if applicable).
	ResetAfter time.Duration

	// RetryAfter suggests when to retry if the request was denied.
	RetryAfter time.Duration
}
