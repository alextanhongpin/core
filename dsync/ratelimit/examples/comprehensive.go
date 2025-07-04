// Package main demonstrates comprehensive rate limiting patterns
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Setup Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Clean up for demo
	client.FlushAll(context.Background())

	fmt.Println("=== Rate Limiting Examples ===\n")

	// 1. Basic Fixed Window
	demonstrateFixedWindow(client)

	// 2. GCRA with burst
	demonstrateGCRA(client)

	// 3. Detailed rate limit checks
	demonstrateDetailedChecks(client)

	// 4. Rate limiting patterns
	demonstrateRateLimitingPatterns(client)

	// 5. Performance comparison
	demonstratePerformanceComparison(client)

	// 6. Error handling
	demonstrateErrorHandling(client)

	fmt.Println("\n=== All Examples Complete ===")
}

func demonstrateFixedWindow(client *redis.Client) {
	fmt.Println("1. Fixed Window Rate Limiting")
	fmt.Println("   Limit: 5 requests per 2 seconds")

	rl := ratelimit.NewFixedWindow(client, 5, 2*time.Second)
	ctx := context.Background()
	key := "demo:fixed-window"

	fmt.Println("   Making 10 requests...")
	for i := 0; i < 10; i++ {
		allowed, err := rl.Allow(ctx, key)
		if err != nil {
			log.Printf("   Error: %v", err)
			continue
		}

		remaining, _ := rl.Remaining(ctx, key)
		resetAfter, _ := rl.ResetAfter(ctx, key)

		status := "✅ ALLOWED"
		if !allowed {
			status = "❌ DENIED"
		}

		fmt.Printf("   Request %d: %s (remaining: %d, reset in: %v)\n",
			i+1, status, remaining, resetAfter.Truncate(time.Millisecond))

		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println()
}

func demonstrateGCRA(client *redis.Client) {
	fmt.Println("2. GCRA Rate Limiting")
	fmt.Println("   Limit: 3 requests per second, burst: 2")

	rl := ratelimit.NewGCRA(client, 3, time.Second, 2)
	ctx := context.Background()
	key := "demo:gcra"

	fmt.Println("   Making burst requests then steady...")
	requests := []time.Duration{
		0,                       // Burst 1
		50 * time.Millisecond,   // Burst 2
		100 * time.Millisecond,  // Burst 3 (should be denied)
		333 * time.Millisecond,  // Steady rate
		666 * time.Millisecond,  // Steady rate
		1000 * time.Millisecond, // Next second
	}

	start := time.Now()
	for i, delay := range requests {
		time.Sleep(delay - time.Since(start) + start.Sub(start))

		allowed, err := rl.Allow(ctx, key)
		if err != nil {
			log.Printf("   Error: %v", err)
			continue
		}

		elapsed := time.Since(start)
		status := "✅ ALLOWED"
		if !allowed {
			status = "❌ DENIED"
		}

		fmt.Printf("   Request %d at %v: %s\n",
			i+1, elapsed.Truncate(time.Millisecond), status)
	}
	fmt.Println()
}

func demonstrateDetailedChecks(client *redis.Client) {
	fmt.Println("3. Detailed Rate Limit Checks")

	rl := ratelimit.NewFixedWindow(client, 3, 5*time.Second)
	ctx := context.Background()
	key := "demo:detailed"

	fmt.Println("   Using Check() method for detailed information...")

	for i := 0; i < 5; i++ {
		result, err := rl.Check(ctx, key)
		if err != nil {
			log.Printf("   Error: %v", err)
			continue
		}

		status := "✅ ALLOWED"
		if !result.Allowed {
			status = "❌ DENIED"
		}

		fmt.Printf("   Check %d: %s\n", i+1, status)
		fmt.Printf("     - Remaining: %d\n", result.Remaining)
		fmt.Printf("     - Reset after: %v\n", result.ResetAfter.Truncate(time.Millisecond))
		fmt.Printf("     - Retry after: %v\n", result.RetryAfter.Truncate(time.Millisecond))

		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println()
}

func demonstrateRateLimitingPatterns(client *redis.Client) {
	fmt.Println("4. Common Rate Limiting Patterns")

	// Per-user rate limiting
	fmt.Println("   Per-User Rate Limiting:")
	userRL := ratelimit.NewFixedWindow(client, 10, time.Minute)
	ctx := context.Background()

	users := []string{"user:123", "user:456", "user:789"}
	for _, userID := range users {
		allowed, _ := userRL.Allow(ctx, userID)
		remaining, _ := userRL.Remaining(ctx, userID)
		fmt.Printf("     %s: allowed=%t, remaining=%d\n", userID, allowed, remaining)
	}

	// API key rate limiting with different tiers
	fmt.Println("   API Key Tiers:")
	premiumRL := ratelimit.NewGCRA(client, 1000, time.Hour, 100)
	standardRL := ratelimit.NewGCRA(client, 100, time.Hour, 10)
	basicRL := ratelimit.NewFixedWindow(client, 10, time.Hour)

	apiKeys := map[string]ratelimit.RateLimiter{
		"premium:abc123":  premiumRL,
		"standard:def456": standardRL,
		"basic:ghi789":    basicRL,
	}

	for apiKey, rl := range apiKeys {
		allowed, _ := rl.Allow(ctx, apiKey)
		fmt.Printf("     %s: allowed=%t\n", apiKey, allowed)
	}

	// Bulk operations
	fmt.Println("   Bulk Operations:")
	bulkRL := ratelimit.NewGCRA(client, 100, time.Minute, 20)
	batchSizes := []int{5, 15, 25, 50}

	for _, size := range batchSizes {
		allowed, _ := bulkRL.AllowN(ctx, "batch:processor", size)
		fmt.Printf("     Batch size %d: allowed=%t\n", size, allowed)
	}
	fmt.Println()
}

func demonstratePerformanceComparison(client *redis.Client) {
	fmt.Println("5. Performance Comparison")

	ctx := context.Background()
	iterations := 100

	// Fixed Window performance
	fmt.Println("   Fixed Window Performance:")
	fwRL := ratelimit.NewFixedWindow(client, 1000000, time.Hour) // High limit

	start := time.Now()
	for i := 0; i < iterations; i++ {
		fwRL.Allow(ctx, fmt.Sprintf("perf:fw:%d", i))
	}
	fwDuration := time.Since(start)
	fmt.Printf("     %d requests in %v (%.2f req/sec)\n",
		iterations, fwDuration, float64(iterations)/fwDuration.Seconds())

	// GCRA performance
	fmt.Println("   GCRA Performance:")
	gcraRL := ratelimit.NewGCRA(client, 1000000, time.Hour, 1000)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		gcraRL.Allow(ctx, fmt.Sprintf("perf:gcra:%d", i))
	}
	gcraDuration := time.Since(start)
	fmt.Printf("     %d requests in %v (%.2f req/sec)\n",
		iterations, gcraDuration, float64(iterations)/gcraDuration.Seconds())

	fmt.Printf("   Performance ratio: %.2fx\n", fwDuration.Seconds()/gcraDuration.Seconds())
	fmt.Println()
}

func demonstrateErrorHandling(client *redis.Client) {
	fmt.Println("6. Error Handling Patterns")

	rl := ratelimit.NewFixedWindow(client, 5, time.Minute)
	ctx := context.Background()

	// Simulate Redis connection error by using wrong key
	fmt.Println("   Graceful degradation on errors:")

	// This would normally cause an error, but we'll simulate it
	allowed, err := rateLimitWithFallback(ctx, rl, "test:key")
	fmt.Printf("     Allowed with fallback: %t (error: %v)\n", allowed, err)

	// Circuit breaker pattern
	fmt.Println("   Circuit breaker pattern:")
	cb := &CircuitBreaker{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
	}

	for i := 0; i < 5; i++ {
		allowed := rateLimitWithCircuitBreaker(ctx, rl, "cb:key", cb)
		fmt.Printf("     Request %d: allowed=%t, state=%s\n", i+1, allowed, cb.State())
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
}

// Helper functions for error handling examples

func rateLimitWithFallback(ctx context.Context, rl ratelimit.RateLimiter, key string) (bool, error) {
	allowed, err := rl.Allow(ctx, key)
	if err != nil {
		// Log error and fail open (allow request)
		log.Printf("Rate limit check failed: %v", err)
		return true, nil // Graceful degradation
	}
	return allowed, nil
}

type CircuitBreaker struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	failures         int
	lastFailureTime  time.Time
	state            string
	mu               sync.Mutex
}

func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "" {
		cb.state = "CLOSED"
	}

	// Reset if timeout passed
	if cb.state == "OPEN" && time.Since(cb.lastFailureTime) > cb.ResetTimeout {
		cb.state = "HALF_OPEN"
		cb.failures = 0
	}

	return cb.state
}

func rateLimitWithCircuitBreaker(ctx context.Context, rl ratelimit.RateLimiter, key string, cb *CircuitBreaker) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check circuit breaker state
	if cb.State() == "OPEN" {
		return false // Circuit is open, deny all requests
	}

	allowed, err := rl.Allow(ctx, key)
	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.failures >= cb.FailureThreshold {
			cb.state = "OPEN"
		}

		return false // Deny on error
	}

	// Reset failures on success
	if cb.state == "HALF_OPEN" {
		cb.state = "CLOSED"
		cb.failures = 0
	}

	return allowed
}
