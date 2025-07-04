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
	// Allow checks if a single request is allowed for the given key.
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if N requests are allowed for the given key.
	AllowN(ctx context.Context, key string, n int) (bool, error)
}

// WindowRateLimiter extends RateLimiter with window-based information.
type WindowRateLimiter interface {
	RateLimiter

	// Remaining returns the number of requests remaining in the current window.
	Remaining(ctx context.Context, key string) (int, error)

	// ResetAfter returns the duration until the rate limit resets.
	ResetAfter(ctx context.Context, key string) (time.Duration, error)
}

// Result contains detailed information about a rate limit check.
type Result struct {
	// Allowed indicates if the request was allowed.
	Allowed bool

	// Remaining is the number of requests remaining (if applicable).
	Remaining int

	// ResetAfter is the duration until the limit resets (if applicable).
	ResetAfter time.Duration

	// RetryAfter suggests when to retry if the request was denied.
	RetryAfter time.Duration
}

// DetailedRateLimiter provides detailed rate limiting information.
type DetailedRateLimiter interface {
	// Check performs a rate limit check and returns detailed information.
	Check(ctx context.Context, key string) (*Result, error)

	// CheckN performs a rate limit check for N requests and returns detailed information.
	CheckN(ctx context.Context, key string, n int) (*Result, error)
}
