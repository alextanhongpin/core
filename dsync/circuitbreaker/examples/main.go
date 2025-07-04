package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Check command line arguments
	if len(os.Args) > 1 && os.Args[1] == "http" {
		runHTTPServiceDemo()
		return
	}

	// Run basic demo by default
	runBasicDemo()
}

func runBasicDemo() {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create circuit breaker
	cb, stop := circuitbreaker.New(client, "example-service")
	defer stop()

	// Configure circuit breaker for demo
	cb.FailureThreshold = 3
	cb.BreakDuration = 5 * time.Second
	cb.SamplingDuration = 10 * time.Second

	fmt.Println("Circuit Breaker Demo")
	fmt.Println("===================")

	// Simulate service calls
	for i := 0; i < 15; i++ {
		err := cb.Do(ctx, simulateService)

		status := cb.Status()
		fmt.Printf("Call %d: Status=%s, Error=%v\n", i+1, status, err)

		// Add delay between calls
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nWaiting for circuit to recover...")
	time.Sleep(6 * time.Second)

	// Test recovery
	for i := 0; i < 5; i++ {
		err := cb.Do(ctx, func() error {
			fmt.Printf("Recovery call %d: Success\n", i+1)
			return nil
		})

		status := cb.Status()
		fmt.Printf("Recovery %d: Status=%s, Error=%v\n", i+1, status, err)

		time.Sleep(500 * time.Millisecond)
	}
}

// simulateService simulates a service that fails randomly
func simulateService() error {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Fail 70% of the time to trigger circuit breaker
	if rand.Float64() < 0.7 {
		return fmt.Errorf("service failure")
	}

	return nil
}
