# Retry

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/retry.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry)
[![Go Report Card](https://goreportcard.com/badge/github.com/alextanhongpin/core/sync/retry)](https://goreportcard.com/report/github.com/alextanhongpin/core/sync/retry)

A robust, production-ready Go retry package that provides intelligent retry mechanisms with configurable backoff strategies and throttling capabilities. Designed for real-world applications where reliability and performance are critical.

## Features

- **Multiple Backoff Strategies**: Constant, linear, and exponential backoff with jitter
- **Context-Aware**: Full support for context cancellation and timeouts
- **Throttling Support**: Built-in adaptive throttling to prevent resource exhaustion
- **Context-Aware Cancellation**: Full context support for timeouts and cancellation
- **Adaptive Throttling**: Token bucket algorithm to prevent resource exhaustion
- **Iterator API**: Clean Go 1.23+ iterators (Try, Do, DoValue) for structured retry logic
- **HTTP Integration**: Automatic HTTP retries via RoundTripper for transient errors
- **Metrics & Observability**: Prometheus integration and atomic metrics collection

## Installation

```
go get github.com/alextanhongpin/core/sync/retry
```

## Quick Start

### Basic Retry with Default Settings

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	ctx := context.Background()

	// Simple retry with default exponential backoff (base: 1s, cap: 1m)
	err := retry.Do(ctx, func(ctx context.Context) error {
		return callService()
	}, retry.N(5)) // Max 5 attempts

	if err != nil {
		log.Printf("Call failed after retries: %v", err)
	}
}

func callService() error {
	// Your potentially failing operation
	return errors.ErrUnsupported
}
```

### Retry with Value Return

```go
func main() {
	ctx := context.Background()

	// DoValue returns the value when successful, or the final error
	user, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
		return callService()
	}, retry.N(3))

	if err != nil {
		log.Printf("Call failed after retries: %v", err)
		return
	}

	log.Printf("Got response: %s", user)
}

func callService() (string, error) {
	return fmt.Sprintf("response"), nil
}
```

## Configuration Options

### Attempt Count

Control how many times an operation will be attempted:

```go
// Using the N helper (shorthand for WithAttempts)
r := retry.New(retry.N(3)) // Max 3 retries after the initial attempt (up to 4 total)

// Or using constant shorthand
r := retry.New(retry.NoWait, retry.Throttle(), retry.N(3)) // No backoff + throttle
```

### Backoff Strategies

Choose from built-in backoff strategies or implement custom ones:

#### Exponential Backoff with Jitter (Recommended for Production)

// Exponential backoff with jitter: base * 2^attempts (randomized)

```go
// Configure exponential backoff with 100ms base and 30s cap
r := retry.New(retry.Exponential(100*time.Millisecond, 30*time.Second))

err := r.Do(ctx, func(ctx context.Context) error {
	return callExternalService()
}, 5) // Or use N(5) for attempt count
```

#### Constant Backoff for Predictable Timing

```go
// No delay: attempts are made immediately if they fail
r := retry.New(retry.NoWait, retry.N(3))

err := r.Do(ctx, func(ctx context.Context) error {
	return processMessage()
}, 5)
```

#### Linear Backoff for Gradual Increase

```go
// Linearly increasing delays: 0s, 1s, 2s, 3s...
// Formula: Period * attempt_number (where attempt_number starts at 0)

err := r.Do(ctx, func(ctx context.Context) error {
	return uploadFile()
}, 5)
```

### Throttling for Rate-Limited APIs

Use adaptive throttling to prevent overwhelming downstream services:

```go
// Configure throttling to prevent resource exhaustion
r := retry.New(retry.Throttle()) // Uses default settings: MaxTokens=10, TokenRatio=0.1

err := r.Do(ctx, func(ctx context.Context) error {
	return callRateLimitedAPI()
}, 20)
```

Handle throttle errors explicitly when needed:

```go
if err != nil {
    if errors.Is(err, retry.ErrThrottled) {
        log.Println("Operation was throttled, try again later")
    } else if errors.Is(err, retry.ErrLimitExceeded) {
        log.Println("Maximum retry attempts exceeded")
    } else {
        log.Printf("Operation failed: %v", err)
    }
}
```

### Custom Throttler Configuration

Fine-tune the token bucket algorithm:

```go
throttlerOpts := &retry.ThrottlerOptions{
	MaxTokens:  10,   // Token bucket size
	TokenRatio: 0.2,  // Token replenishment rate per success
}

r := retry.New().WithThrottler(retry.NewThrottler(throttlerOpts))
```

## Iterator API

The iterator-based approach gives you fine-grained control over retry logic:

```go
func main() {
	ctx := context.Background()
	r := retry.New(retry.NoWait, retry.Throttle(), retry.N(3))

	var lastError error
	for attempt, retryErr := range r.Try(ctx, 5) {
		if retryErr != nil {
			log.Printf("Retry stopped: %v (after %d attempts)", retryErr, attempt)
			break
		}

		log.Printf("Attempt %d starting...", attempt+1)

		err := performComplexOperation()
		if err == nil {
			log.Println("Operation succeeded!")
			break
		}

		lastError = err
	}

	if lastError != nil {
		log.Printf("Final error: %v", lastError)
	}
}

func performComplexOperation() error {
	return errors.ErrUnsupported
}
```

## HTTP Client Integration

### Automatic HTTP Retry with Custom Status Codes

Configure which HTTP status codes should trigger automatic retries:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	retryableStatusCodes := []int{
		http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	statusCodeFunc := func(code int) error {
		if errors.Is(retryErr := fmt.Errorf("retryable status: %d", code), nil) {
			return fmt.Errorf("%d: %s", code, http.StatusText(code))
		}
		return nil
	}

	retryTransport := retry.NewRoundTripper(
		http.DefaultTransport,
		retry.New(), // Your retry handler
	).With(&retry.RoundTripper{
		StatusCode: statusCodeFunc,
		MaxRetries: 10,
	})

	client := &http.Client{
		Transport: retryTransport,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Get("https://api.example.com/data")
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Success! Status: %d\n", resp.StatusCode)
}
```

## Metrics and Observability

### Atomic Metrics Collection

The package includes built-in atomic metrics collectors for tracking retry behavior:

```go
type RetryMetrics struct {
	Attempts      int64
	Successes     int64
	Failures      int64
	Throttles     int64
	LimitExceeded int64
}
```

### Prometheus Integration

Monitor retries via Prometheus metrics endpoint:

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusRetryMetrics() *retry.PrometheusRetryMetricsCollector {
	attempts := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_attempts_total",
		Help: "Total retry attempts.",
	})
	successes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_successes_total",
		Help: "Total retry successes.",
	})
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_failures_total",
		Help: "Total retry failures.",
	})
	throttles := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_throttles_total",
		Help: "Total retry throttles.",
	})
	limitExceeded := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_limit_exceeded_total",
		Help: "Total retry limit exceeded events.",
	})

	prometheus.MustRegister(attempts, successes, failures, throttles, limitExceeded)

	return &retry.PrometheusRetryMetricsCollector{
		Attempts:      attempts,
		Successes:     successes,
		Failures:      failures,
		Throttles:     throttles,
		LimitExceeded: limitExceeded,
	}
}

func main() {
	metrics := newPrometheusRetryMetrics()
	r := retry.New(retry.WithMetricsCollector(metrics), retry.N(3))

	ctx := context.Background()
	_ = r.Do(ctx, func(ctx context.Context) error {
		return errors.ErrUnsupported // Simulate failure for demo
	})

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus retry metrics available at http://localhost:2116/metrics")
	log.Fatal(http.ListenAndServe(":2116", nil))
}
```

## Error Handling

The package provides specific error types for different failure scenarios:

```go
func handleRetryErrors(err error) {
	switch {
	case errors.Is(err, retry.ErrLimitExceeded):
		// Maximum retry attempts reached
		log.Error("Retry limit exceeded, operation failed permanently")

	case errors.Is(err, retry.ErrThrottled):
		// Operation was throttled due to rate limiting
		log.Warn("Operation throttled, try again later")

	case errors.Is(err, context.DeadlineExceeded):
		// Context timeout reached
		log.Error("Operation timed out")

	case errors.Is(err, context.Canceled):
		// Context was cancelled
		log.Info("Operation cancelled")

	default:
		// Other application-specific errors
		log.Error("Operation failed: %v", err)
	}
}
```

## Real-World Examples

### Database Operations with Circuit Breaker Pattern

```go
func main() {
	ctx := context.Background()

	// Configure exponential backoff with short intervals for DB operations
	r := retry.New(retry.Exponential(50*time.Millisecond, 2*time.Second))

	// Add throttling to prevent database overload
	r = retry.WithThrottler(r).With(&retry.ThrottlerOptions{
		MaxTokens:   5,
		TokenRatio:  0.1,
	})

	user, err := retry.DoValue(ctx, func(ctx context.Context) (*User, error) {
		return getUserFromDatabase(ctx, "user123")
	}, retry.N(3))

	if err != nil {
		log.Printf("Database query failed: %v", err)
		return
	}

	log.Printf("Retrieved user: %+v", user)
}

type User struct {
	ID   string
	Name string
}

func getUserFromDatabase(ctx context.Context, userID string) (*User, error) {
	// Simulate database query that might fail
	return &User{ID: userID, Name: "John Doe"}, nil
}
```

### Microservice Communication

```go
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r := retry.New(retry.Exponential(100*time.Millisecond, 10*time.Second))

	response, err := retry.DoValue(ctx, func(ctx context.Context) (*ServiceResponse, error) {
		return callMicroservice(ctx, "process-order", map[string]interface{}{
			"order_id": "12345",
			"amount":   99.99,
		})
	}, retry.N(5))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Service call timed out")
		} else if errors.Is(err, retry.ErrLimitExceeded) {
			log.Println("Exceeded maximum retry attempts")
		} else {
			log.Printf("Service call failed: %v", err)
		}
		return
	}

	log.Printf("Service response: %+v", response)
}

type ServiceResponse struct {
	Status string
	Data   map[string]interface{}
}

func callMicroservice(ctx context.Context, endpoint string, payload map[string]interface{}) (*ServiceResponse, error) {
	// Simulate microservice call
	return &ServiceResponse{
		Status: "success",
		Data:   payload,
	}, nil
}
```

### Batch Processing with Error Handling

```go
func main() {
	ctx := context.Background()
	items := []string{"item1", "item2", "item3", "item4", "item5"}

	r := retry.New(retry.NoWait, retry.N(3))

	var successCount, failureCount int

	for _, item := range items {
		err := r.Do(ctx, func(ctx context.Context) error {
			return processItem(item)
		}, retry.N(3))

		if err != nil {
			log.Printf("Failed to process %s: %v", item, err)
			failureCount++
		} else {
			log.Printf("Successfully processed %s", item)
			successCount++
		}
	}

	log.Printf("Batch complete: %d/%d succeeded", successCount, len(items))
}

func processItem(item string) error {
	// Simulate item processing with occasional failures
	if rand.Float64() < 0.3 {
		return errors.ErrUnsupported
	}
	return nil
}
```

### Concurrent Retry Operations

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	r := retry.New(retry.NoWait, retry.Throttle(), retry.N(3))

	var mu sync.Mutex
	counter := make(map[int]int)

	ctx := context.Background()

	var failed, success int64
	var skipped int64
	n := 100

	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			var i int
			err := r.Do(ctx, func(context.Context) error {
				i++

				mu.Lock()
				counter[i]++
				mu.Unlock()

				if rand.Float64() < 0.2 {
					return errors.ErrUnsupported
				}

				return nil
			})

			if errors.Is(err, retry.ErrThrottled) {
				skipped++
			}
			if errors.Is(err, retry.ErrLimitExceeded) {
				failed++
			}
			if err == nil {
				success++
			}
		})
	}

	wg.Wait()
	fmt.Println("success:", success)
	fmt.Println("skipped:", skipped)
	fmt.Println("failed:", failed)
	fmt.Println("counter:", counter)
	var retries int
	for k, v := range counter {
		if k > 0 {
			retries += v
		}
	}
	fmt.Println("retries:", retries)
}
```

## Error Types

| Error | Description |
|-------|-------------|
| retry.ErrLimitExceeded | Maximum retry attempts reached |
| retry.ErrThrottled | Operation was throttled due to rate limiting |
| context.DeadlineExceeded | Context timeout reached |
| context.Canceled | Context was cancelled |

## Performance Considerations

### Memory Usage

- The iterator-based API minimizes memory allocations
- Throttler uses efficient token bucket algorithm
- No goroutine leaks in concurrent scenarios

### CPU Usage

- Exponential backoff uses efficient random jitter calculation
- Context cancellation is checked before each retry attempt
- Minimal overhead for successful operations

### Network Efficiency

- Jittered exponential backoff reduces thundering herd problems
- Adaptive throttling prevents overwhelming downstream services
- Configurable timeout support for different operation types

## Best Practices

1. Use Context Timeouts: Always use context with appropriate timeouts
2. Choose Appropriate Backoff: Exponential for external services, constant for internal operations
3. Configure Throttling: Enable throttling for rate-limited APIs
4. Monitor Metrics: Track retry attempts and success rates using Prometheus or custom collectors
5. Set Reasonable Limits: Balance between reliability and performance
6. Handle Specific Errors: Differentiate between temporary and permanent failures

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
