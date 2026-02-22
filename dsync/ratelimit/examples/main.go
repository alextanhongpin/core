// Package main demonstrates comprehensive rate limiting patterns
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

func main() {
	stop := redistest.Init()
	defer stop()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	_ = rdb.FlushDB(ctx).Err()
	defer rdb.Close()
	ratelimit.Setup(ctx, rdb)

	fmt.Println("=== Rate Limiting Examples ===\n")

	// 1. Basic demonstrations
	demonstrateBasicUsage(ctx, rdb)

	// 2. Advanced patterns
	demonstrateAdvancedPatterns(ctx, rdb)

	// 3. Performance comparison
	demonstratePerformanceComparison(ctx, rdb)

	// 4. LLM specific examples
	demonstrateLLMRateLimits(ctx, rdb)

	fmt.Println("\n=== All Examples Complete ===")
}

func demonstrateBasicUsage(ctx context.Context, client *redis.Client) {
	fmt.Println("1. Basic Rate Limiting Usage")

	// Fixed Window Example
	fmt.Println("   Fixed Window (5 requests per 2 seconds):")
	fw := ratelimit.NewFixedWindow(client, 5, 2*time.Second)

	for i := 0; i < 8; i++ {
		r, _ := fw.Limit(ctx, "user:123")

		status := "✅"
		if !r.Allow {
			status = "❌"
		}

		fmt.Printf("     Request %d: %s (remaining: %d, reset: %v)\n",
			i+1, status, r.Remaining, r.ResetAfter.Truncate(time.Millisecond))

		if i == 4 { // After hitting limit, wait for reset
			time.Sleep(2 * time.Second)
		}
	}

	// GCRA Example
	fmt.Println("   GCRA (3 req/sec, burst=2):")
	gcra := ratelimit.NewGCRA(client, 3, time.Second, 2)

	// Burst requests
	for i := 0; i < 3; i++ {
		allowed, _ := gcra.Allow(ctx, "stream:1")
		status := "✅"
		if !allowed {
			status = "❌"
		}
		fmt.Printf("     Burst %d: %s\n", i+1, status)
	}

	// Wait and try steady rate
	time.Sleep(334 * time.Millisecond) // 1/3 second
	allowed, _ := gcra.Allow(ctx, "stream:1")
	status := "✅"
	if !allowed {
		status = "❌"
	}
	fmt.Printf("     Steady: %s\n", status)
	fmt.Println()
}

func demonstrateAdvancedPatterns(ctx context.Context, client *redis.Client) {
	fmt.Println("2. Advanced Rate Limiting Patterns")

	// Multi-tier API limits
	fmt.Println("   Multi-tier API Limits:")

	tiers := map[string]ratelimit.RateLimiter{
		"premium":  ratelimit.NewGCRA(client, 1000, time.Hour, 100),
		"standard": ratelimit.NewGCRA(client, 100, time.Hour, 10),
		"basic":    ratelimit.NewFixedWindow(client, 10, time.Hour),
	}

	for tier, rl := range tiers {
		allowed, _ := rl.Allow(ctx, fmt.Sprintf("api:%s:key123", tier))
		fmt.Printf("     %s tier: %s\n", tier, map[bool]string{true: "✅", false: "❌"}[allowed])
	}

	// Bulk operations
	fmt.Println("   Bulk Operations:")
	bulkRL := ratelimit.NewGCRA(client, 100, time.Minute, 20)

	batches := []int{5, 15, 30, 50}
	for _, size := range batches {
		allowed, _ := bulkRL.AllowN(ctx, "batch:processor", size)
		status := "✅"
		if !allowed {
			status = "❌"
		}
		fmt.Printf("     Batch size %d: %s\n", size, status)
	}

	// Detailed checks
	fmt.Println("   Detailed Rate Limit Information:")
	detailRL := ratelimit.NewFixedWindow(client, 3, 10*time.Second)

	for i := 0; i < 4; i++ {
		result, _ := detailRL.Limit(ctx, "detail:check")
		status := "✅"
		if !result.Allow {
			status = "❌"
		}
		fmt.Printf("     Check %d: %s (remaining: %d, retry in: %v)\n",
			i+1, status, result.Remaining, result.RetryAfter.Truncate(time.Millisecond))
	}
	fmt.Println()
}

func demonstratePerformanceComparison(ctx context.Context, client *redis.Client) {
	fmt.Println("3. Performance Comparison")

	iterations := 50

	// Fixed Window
	fw := ratelimit.NewFixedWindow(client, 1000000, time.Hour)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		fw.Allow(ctx, fmt.Sprintf("perf:fw:%d", i))
	}
	fwDuration := time.Since(start)

	// GCRA
	gcra := ratelimit.NewGCRA(client, 1000000, time.Hour, 1000)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		gcra.Allow(ctx, fmt.Sprintf("perf:gcra:%d", i))
	}
	gcraDuration := time.Since(start)

	fmt.Printf("   Fixed Window: %d ops in %v (%.1f ops/sec)\n",
		iterations, fwDuration, float64(iterations)/fwDuration.Seconds())
	fmt.Printf("   GCRA: %d ops in %v (%.1f ops/sec)\n",
		iterations, gcraDuration, float64(iterations)/gcraDuration.Seconds())

	ratio := fwDuration.Seconds() / gcraDuration.Seconds()
	if ratio > 1 {
		fmt.Printf("   GCRA is %.1fx faster\n", ratio)
	} else {
		fmt.Printf("   Fixed Window is %.1fx faster\n", 1/ratio)
	}
}

// demonstrateLLMRateLimits shows how to limit LLM usage by request count and
// token count, both per minute and per day. It uses fixed‑window rate
// limiters and demonstrates combined checks.
func demonstrateLLMRateLimits(ctx context.Context, client *redis.Client) {
	fmt.Println("4. LLM Rate Limiting Examples")

	// Request limits
	reqPerMin := ratelimit.NewFixedWindow(client, 60, time.Minute)
	reqPerDay := ratelimit.NewFixedWindow(client, 1_000, 24*time.Hour)

	// Token limits – example values
	tokenPerMin := ratelimit.NewFixedWindow(client, 5_000, time.Minute)
	tokenPerDay := ratelimit.NewFixedWindow(client, 200_000, 24*time.Hour)

	// Simulate 10 requests; each consumes a deterministic token amount
	for i := 1; i <= 100; i++ {
		tokenCount := 500 + i*20 // deterministic for reproducibility
		allowedReq, _ := reqPerMin.Allow(ctx, "llm:request:123")
		allowedToken, _ := tokenPerMin.AllowN(ctx, "llm:token:123", tokenCount)
		allowedReqDay, _ := reqPerDay.Allow(ctx, "llm:request:123")
		allowedTokenDay, _ := tokenPerDay.AllowN(ctx, "llm:token:123", tokenCount)

		status := "✅"
		if !(allowedReq && allowedToken && allowedReqDay && allowedTokenDay) {
			status = "❌"
		}
		fmt.Printf("   Request %d: %s (tokens %d) – req/min %t, token/min %t, req/day %t, token/day %t\n",
			i, status, tokenCount, allowedReq, allowedToken, allowedReqDay, allowedTokenDay)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
}

// Legacy functions for backward compatibility
type ratelimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

func simulate(ctx context.Context, rl ratelimiter, key string) error {
	fmt.Printf("Simulating traffic for %s...\n", key)

	count := 0
	for i := 0; i < 10; i++ {
		allowed, err := rl.Allow(ctx, key)
		if err != nil {
			return err
		}
		if allowed {
			count++
		}
		fmt.Printf("  Request %d: %s\n", i+1, map[bool]string{true: "✅", false: "❌"}[allowed])
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Total allowed: %d/10\n", count)
	return nil
}

func newFixedWindow(client *redis.Client) *ratelimit.FixedWindow {
	return ratelimit.NewFixedWindow(client, 5, time.Second)
}

func newGCRA(client *redis.Client) *ratelimit.GCRA {
	return ratelimit.NewGCRA(client, 5, time.Second, 1)
}
