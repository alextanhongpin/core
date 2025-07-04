// Simple circuit breaker example showing basic usage
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

// UnreliableService simulates a service that can fail
type UnreliableService struct {
	failureRate float64
}

func (s *UnreliableService) Call(ctx context.Context, data string) (string, error) {
	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)

	// Randomly fail based on failure rate
	if rand.Float64() < s.failureRate {
		return "", errors.New("service temporarily unavailable")
	}

	return fmt.Sprintf("Processed: %s", data), nil
}

func main() {
	fmt.Println("🔌 Simple Circuit Breaker Example")
	fmt.Println("=================================")

	// Create a circuit breaker with default settings
	cb := circuitbreaker.New()

	// Configure for quick demonstration
	cb.BreakDuration = 1 * time.Second
	cb.FailureThreshold = 3
	cb.FailureRatio = 0.5

	// Add a state change callback
	cb.OnStateChange = func(old, new circuitbreaker.Status) {
		fmt.Printf("🔄 Circuit breaker: %s -> %s\n", old, new)
	}

	// Create a service that fails 80% of the time initially
	service := &UnreliableService{failureRate: 0.8}

	fmt.Println("\n🚀 Making calls to unreliable service...")

	// Make 10 calls
	for i := 1; i <= 10; i++ {
		err := cb.Do(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := service.Call(ctx, fmt.Sprintf("request-%d", i))
			if err == nil {
				fmt.Printf("Call %d: ✅ %s\n", i, result)
			}
			return err
		})

		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			fmt.Printf("Call %d: ⚡ Circuit breaker is open - request blocked\n", i)
		} else if err != nil {
			fmt.Printf("Call %d: ❌ %v\n", i, err)
		}

		// Small delay between calls
		time.Sleep(200 * time.Millisecond)
	}

	// Show circuit breaker recovery
	fmt.Println("\n🌟 Improving service reliability...")
	service.failureRate = 0.1 // Much better now

	// Wait for circuit to try half-open
	time.Sleep(cb.BreakDuration + 100*time.Millisecond)

	// Make a few more calls
	for i := 11; i <= 15; i++ {
		err := cb.Do(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := service.Call(ctx, fmt.Sprintf("request-%d", i))
			if err == nil {
				fmt.Printf("Call %d: ✅ %s\n", i, result)
			}
			return err
		})

		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			fmt.Printf("Call %d: ⚡ Circuit breaker is open - request blocked\n", i)
		} else if err != nil {
			fmt.Printf("Call %d: ❌ %v\n", i, err)
		}

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("\n✅ Final circuit breaker status: %s\n", cb.Status())
}
