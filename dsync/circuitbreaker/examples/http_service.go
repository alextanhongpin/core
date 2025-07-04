package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/redis/go-redis/v9"
)

// HTTPService demonstrates circuit breaker with HTTP calls
type HTTPService struct {
	client *http.Client
	cb     *circuitbreaker.CircuitBreaker
}

func NewHTTPService(redisClient *redis.Client, serviceName string) *HTTPService {
	// Create circuit breaker with custom configuration
	config := circuitbreaker.Config{
		BreakDuration:    10 * time.Second,
		FailureThreshold: 5,
		FailureRatio:     0.6,
		SamplingDuration: 30 * time.Second,
		SuccessThreshold: 3,
		// Custom failure counting for HTTP errors
		FailureCount: func(err error) int {
			if errors.Is(err, context.DeadlineExceeded) {
				return 3 // Timeout errors are more severe
			}
			return 1
		},
		// Custom slow call detection
		SlowCallCount: func(duration time.Duration) int {
			if duration > 2*time.Second {
				return 2 // Penalize very slow calls
			}
			return 0
		},
	}

	cb, _ := circuitbreaker.NewWithConfig(redisClient, serviceName, config)

	return &HTTPService{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		cb: cb,
	}
}

func (s *HTTPService) Get(ctx context.Context, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	cbErr := s.cb.Do(ctx, func() error {
		resp, err = s.client.Get(url)
		if err != nil {
			return err
		}

		// Treat 5xx errors as failures
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return nil
	})

	if cbErr != nil {
		switch cbErr {
		case circuitbreaker.ErrUnavailable:
			return nil, fmt.Errorf("service unavailable: circuit breaker is open")
		case circuitbreaker.ErrForcedOpen:
			return nil, fmt.Errorf("service maintenance: circuit breaker is forced open")
		default:
			return nil, cbErr
		}
	}

	return resp, err
}

func (s *HTTPService) Status() circuitbreaker.Status {
	return s.cb.Status()
}

func runHTTPServiceDemo() {
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

	// Create HTTP service with circuit breaker
	service := NewHTTPService(client, "external-api")

	fmt.Println("HTTP Service Circuit Breaker Demo")
	fmt.Println("==================================")

	// URLs to test (some will fail)
	urls := []string{
		"https://httpbin.org/status/200",
		"https://httpbin.org/status/500",
		"https://httpbin.org/status/503",
		"https://httpbin.org/delay/3",
		"https://httpbin.org/status/200",
		"https://invalid-url-that-will-fail.com",
	}

	// Test the service
	for i, url := range urls {
		fmt.Printf("\nCall %d: %s\n", i+1, url)

		resp, err := service.Get(ctx, url)
		status := service.Status()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Success: %s\n", resp.Status)
			resp.Body.Close()
		}

		fmt.Printf("Circuit Status: %s\n", status)

		// Add delay between calls
		time.Sleep(1 * time.Second)
	}

	// Demonstrate forced open
	fmt.Println("\n--- Demonstrating Forced Open ---")
	service.cb.ForceOpen()
	fmt.Printf("Circuit Status: %s\n", service.Status())

	_, err := service.Get(ctx, "https://httpbin.org/status/200")
	if err != nil {
		fmt.Printf("Expected error with forced open: %v\n", err)
	}

	// Demonstrate disabled state
	fmt.Println("\n--- Demonstrating Disabled State ---")
	service.cb.Disable()
	fmt.Printf("Circuit Status: %s\n", service.Status())

	resp, err := service.Get(ctx, "https://httpbin.org/status/200")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success with disabled circuit: %s\n", resp.Status)
		resp.Body.Close()
	}
}
