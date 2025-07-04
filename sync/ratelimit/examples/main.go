package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
	fmt.Println("=== Rate Limiting Examples ===")

	// Example 1: API Gateway Rate Limiting
	fmt.Println("1. API Gateway Rate Limiting")
	apiGatewayExample()

	// Example 2: User-specific Rate Limiting
	fmt.Println("\n2. User-specific Rate Limiting")
	userRateLimitExample()

	// Example 3: Different Algorithm Comparison
	fmt.Println("\n3. Algorithm Comparison")
	algorithmComparisonExample()

	// Example 4: HTTP Middleware
	fmt.Println("\n4. HTTP Middleware Integration")
	httpMiddlewareExample()
}

// Example 1: API Gateway Rate Limiting
func apiGatewayExample() {
	// Create GCRA limiter: 10 requests per second with burst of 5
	limiter, err := ratelimit.NewGCRA(10, time.Second, 5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("API Gateway: 10 req/s with burst of 5\n")

	// Simulate API requests
	for i := 0; i < 15; i++ {
		allowed := limiter.Allow()
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("Request %2d: %s\n", i+1, status)

		// Small delay to show rate limiting effect
		time.Sleep(50 * time.Millisecond)
	}
}

// Example 2: User-specific Rate Limiting
func userRateLimitExample() {
	// Create multi-key GCRA limiter: 5 requests per minute per user
	limiter, err := ratelimit.NewMultiGCRA(5, time.Minute, 2)
	if err != nil {
		log.Fatal(err)
	}

	users := []string{"user1", "user2", "user3"}

	fmt.Printf("Per-user limiting: 5 req/min per user\n")

	// Simulate requests from different users
	for round := 0; round < 3; round++ {
		fmt.Printf("\nRound %d:\n", round+1)
		for _, user := range users {
			for i := 0; i < 3; i++ {
				allowed := limiter.Allow(user)
				status := "✓ ALLOWED"
				if !allowed {
					status = "✗ DENIED"
				}
				fmt.Printf("  %s request %d: %s\n", user, i+1, status)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Example 3: Algorithm Comparison
func algorithmComparisonExample() {
	// Create different rate limiters for comparison
	gcra, _ := ratelimit.NewGCRA(5, time.Second, 2)
	fixedWindow, _ := ratelimit.NewFixedWindow(5, time.Second)
	slidingWindow, _ := ratelimit.NewSlidingWindow(5, time.Second)

	fmt.Printf("Comparing algorithms: 5 req/s\n")

	algorithms := []struct {
		name    string
		limiter interface{ Allow() bool }
	}{
		{"GCRA", gcra},
		{"Fixed Window", fixedWindow},
		{"Sliding Window", slidingWindow},
	}

	for _, alg := range algorithms {
		fmt.Printf("\n%s:\n", alg.name)

		// Test burst behavior
		for i := 0; i < 8; i++ {
			allowed := alg.limiter.Allow()
			status := "✓"
			if !allowed {
				status = "✗"
			}
			fmt.Printf("  %s", status)
		}
		fmt.Println()
	}
}

// Example 4: HTTP Middleware Integration
func httpMiddlewareExample() {
	// Create rate limiter for HTTP middleware
	limiter, err := ratelimit.NewMultiGCRA(10, time.Minute, 5)
	if err != nil {
		log.Fatal(err)
	}

	// Create HTTP middleware
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as key (simplified for demo)
			clientIP := r.RemoteAddr
			if clientIP == "" {
				clientIP = "unknown"
			}

			if !limiter.Allow(clientIP) {
				w.Header().Set("X-RateLimit-Limit", "10")
				w.Header().Set("X-RateLimit-Window", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with middleware
	server := &http.Server{
		Addr:           ":8080",
		Handler:        middleware(handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Printf("HTTP server example configured with rate limiting\n")
	fmt.Printf("Would start server on :8080 with 10 req/min per IP\n")
	fmt.Printf("Middleware adds X-RateLimit-* headers\n")

	// Don't actually start the server in this example
	_ = server
}

// Real-world production example: E-commerce API
func ecommerceAPIExample() {
	ctx := context.Background()

	// Different rate limits for different endpoints
	publicAPI, _ := ratelimit.NewMultiGCRA(100, time.Hour, 10)   // 100/hour for public
	userAPI, _ := ratelimit.NewMultiGCRA(1000, time.Hour, 50)    // 1000/hour for users
	adminAPI, _ := ratelimit.NewMultiGCRA(10000, time.Hour, 100) // 10000/hour for admins

	// Simulate different types of requests
	requests := []struct {
		userType string
		userID   string
		endpoint string
		limiter  *ratelimit.MultiGCRA
	}{
		{"anonymous", "anon1", "/api/products", publicAPI},
		{"user", "user123", "/api/orders", userAPI},
		{"admin", "admin456", "/api/analytics", adminAPI},
	}

	fmt.Printf("E-commerce API Rate Limiting:\n")

	for _, req := range requests {
		allowed := req.limiter.Allow(req.userID)
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ DENIED"
		}
		fmt.Printf("  %s %s -> %s: %s\n",
			req.userType, req.endpoint, req.userID, status)
	}

	_ = ctx // Context for future enhancements
}
