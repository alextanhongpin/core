package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimit creates a simple in-memory rate limiting middleware using a token bucket algorithm.
// This is suitable for single-instance applications. For distributed systems,
// consider using external rate limiting solutions like Redis.
//
// Example:
//
//	// Allow 100 requests per minute per IP
//	handler = middleware.RateLimit(100, time.Minute, middleware.ByIP)(handler)
func RateLimit(requests int, window time.Duration, keyFunc KeyFunc) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(requests, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !limiter.Allow(key) {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
				w.Header().Set("X-RateLimit-Window", window.String())
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Too Many Requests"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// KeyFunc is a function that extracts a rate limiting key from a request.
type KeyFunc func(*http.Request) string

// ByIP extracts the client IP address as the rate limiting key.
func ByIP(r *http.Request) string {
	// Check for forwarded IP first
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

// ByAPIKey extracts an API key from the Authorization header as the rate limiting key.
func ByAPIKey(r *http.Request) string {
	return r.Header.Get("Authorization")
}

// ByUserID extracts a user ID from the request context as the rate limiting key.
// This assumes the user ID has been set in the context by authentication middleware.
func ByUserID(userContextKey string) KeyFunc {
	return func(r *http.Request) string {
		if userID := r.Context().Value(userContextKey); userID != nil {
			return fmt.Sprintf("user:%v", userID)
		}
		return ByIP(r) // Fallback to IP if no user ID
	}
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	requests int
	window   time.Duration
	buckets  map[string]*bucket
	mu       sync.RWMutex
	cleanup  *time.Ticker
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter with the specified rate and window.
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: requests,
		window:   window,
		buckets:  make(map[string]*bucket),
		cleanup:  time.NewTicker(window),
	}

	// Start cleanup goroutine to remove old buckets
	go rl.cleanupOldBuckets()

	return rl
}

// Allow checks if a request is allowed for the given key.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]

	if !exists {
		// Create new bucket
		rl.buckets[key] = &bucket{
			tokens:   rl.requests - 1,
			lastSeen: now,
		}
		return true
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(b.lastSeen)
	tokensToAdd := int(elapsed / rl.window * time.Duration(rl.requests))

	// Refill tokens
	b.tokens = min(rl.requests, b.tokens+tokensToAdd)
	b.lastSeen = now

	// Check if request is allowed
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanupOldBuckets removes buckets that haven't been used recently.
func (rl *RateLimiter) cleanupOldBuckets() {
	for range rl.cleanup.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window * 2) // Keep buckets for 2x the window

		for key, bucket := range rl.buckets {
			if bucket.lastSeen.Before(cutoff) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Close stops the cleanup goroutine.
func (rl *RateLimiter) Close() {
	rl.cleanup.Stop()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
