# Retry

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/retry.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry)
[![Go Report Card](https://goreportcard.com/badge/github.com/alextanhongpin/core/sync/retry)](https://goreportcard.com/report/github.com/alextanhongpin/core/sync/retry)

A robust, production-ready Go retry package that provides intelligent retry mechanisms with configurable backoff strategies and throttling capabilities. Designed for real-world applications where reliability and performance are critical.

## Features

- **Multiple Backoff Strategies**: Constant, linear, and exponential backoff with jitter
- **Context-Aware**: Full support for context cancellation and timeouts
- **Throttling Support**: Built-in adaptive throttling to prevent resource exhaustion
- **Iterator-Based API**: Modern Go 1.23+ iterator interface for flexible retry loops
- **HTTP Integration**: Ready-to-use HTTP round tripper with automatic retry logic
- **Generic Support**: Type-safe retry operations for functions returning values
- **Thread-Safe**: Concurrent operations supported with proper synchronization

## Installation

```bash
go get github.com/alextanhongpin/core/sync/retry
```

## Quick Start

### Basic Retry with Default Settings

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/alextanhongpin/core/sync/retry"
)

func main() {
    ctx := context.Background()
    
    // Simple retry with default exponential backoff
    err := retry.Do(ctx, func(ctx context.Context) error {
        // Your potentially failing operation
        return performAPICall()
    }, 5) // Max 5 attempts
    
    if err != nil {
        fmt.Printf("Operation failed after retries: %v\n", err)
    }
}

func performAPICall() error {
    // Simulate intermittent failures
    return errors.New("temporary network error")
}
```

### Retry with Value Return

```go
func main() {
    ctx := context.Background()
    
    result, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
        return fetchUserData("user123")
    }, 3)
    
    if err != nil {
        fmt.Printf("Failed to fetch user data: %v\n", err)
        return
    }
    
    fmt.Printf("User data: %s\n", result)
}

func fetchUserData(userID string) (string, error) {
    // Simulate database query that might fail
    return "user data", nil
}
```

## Advanced Usage

### Custom Backoff Strategies

#### Exponential Backoff with Jitter (Recommended for Production)

```go
func main() {
    ctx := context.Background()
    
    // Create retry instance with custom exponential backoff
    r := retry.New().WithBackOff(
        retry.NewExponentialBackOff(
            100*time.Millisecond, // Base delay
            30*time.Second,       // Maximum delay cap
        ),
    )
    
    err := r.Do(ctx, func(ctx context.Context) error {
        return callExternalService()
    }, 10)
    
    if err != nil {
        fmt.Printf("Service call failed: %v\n", err)
    }
}
```

#### Constant Backoff for Predictable Timing

```go
func main() {
    ctx := context.Background()
    
    // Fixed delay between retries
    r := retry.New().WithBackOff(
        retry.NewConstantBackOff(500 * time.Millisecond),
    )
    
    err := r.Do(ctx, func(ctx context.Context) error {
        return processMessage()
    }, 5)
    
    if err != nil {
        fmt.Printf("Message processing failed: %v\n", err)
    }
}
```

#### Linear Backoff for Gradual Increase

```go
func main() {
    ctx := context.Background()
    
    // Linearly increasing delays: 1s, 2s, 3s, 4s...
    r := retry.New().WithBackOff(
        retry.NewLinearBackOff(1 * time.Second),
    )
    
    err := r.Do(ctx, func(ctx context.Context) error {
        return uploadFile()
    }, 5)
    
    if err != nil {
        fmt.Printf("File upload failed: %v\n", err)
    }
}
```

### Throttling for Rate-Limited APIs

```go
func main() {
    ctx := context.Background()
    
    // Configure throttling to prevent overwhelming APIs
    throttlerOpts := &retry.ThrottlerOptions{
        MaxTokens:  10,  // Token bucket size
        TokenRatio: 0.2, // Token replenishment rate
    }
    
    r := retry.New()
    r.Throttler = retry.NewThrottler(throttlerOpts)
    
    // This will automatically throttle retries based on success/failure rates
    err := r.Do(ctx, func(ctx context.Context) error {
        return callRateLimitedAPI()
    }, 20)
    
    if err != nil {
        if errors.Is(err, retry.ErrThrottled) {
            fmt.Println("Operation was throttled")
        } else {
            fmt.Printf("Operation failed: %v\n", err)
        }
    }
}
```

### Fine-Grained Control with Iterator API

```go
func main() {
    ctx := context.Background()
    r := retry.New()
    
    var lastError error
    for attempt, retryErr := range r.Try(ctx, 5) {
        if retryErr != nil {
            fmt.Printf("Retry stopped: %v (after %d attempts)\n", retryErr, attempt)
            break
        }
        
        fmt.Printf("Attempt %d\n", attempt+1)
        
        lastError = performComplexOperation()
        if lastError == nil {
            fmt.Println("Operation succeeded!")
            break
        }
        
        // You can add custom logic here between attempts
        logFailureMetrics(attempt, lastError)
    }
}

func performComplexOperation() error {
    // Your complex operation logic
    return nil
}

func logFailureMetrics(attempt int, err error) {
    // Custom logging or metrics collection
    fmt.Printf("Attempt %d failed: %v\n", attempt, err)
}
```

## HTTP Client Integration

### Automatic HTTP Retry with Custom Status Codes

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/alextanhongpin/core/sync/retry"
)

func main() {
    // Create an HTTP client with retry capabilities
    baseTransport := &http.Transport{
        MaxIdleConns:        100,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  true,
    }
    
    retryTransport := retry.NewRoundTripper(
        baseTransport,
        retry.New().WithBackOff(
            retry.NewExponentialBackOff(100*time.Millisecond, 5*time.Second),
        ),
    )
    
    // Configure which status codes should trigger retries
    retryTransport.StatusCode = func(code int) error {
        switch code {
        case http.StatusTooManyRequests,
             http.StatusInternalServerError,
             http.StatusBadGateway,
             http.StatusServiceUnavailable,
             http.StatusGatewayTimeout:
            return fmt.Errorf("retryable status: %d", code)
        }
        return nil
    }
    
    client := &http.Client{
        Transport: retryTransport,
        Timeout:   30 * time.Second,
    }
    
    // This request will automatically retry on transient failures
    resp, err := client.Get("https://api.example.com/data")
    if err != nil {
        fmt.Printf("Request failed: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Printf("Success! Status: %d\n", resp.StatusCode)
}
```

## Real-World Examples

### Database Operations with Circuit Breaker Pattern

```go
func main() {
    ctx := context.Background()
    
    // Configure for database operations
    r := retry.New().WithBackOff(
        retry.NewExponentialBackOff(50*time.Millisecond, 2*time.Second),
    )
    
    // Add throttling to prevent database overload
    r.Throttler = retry.NewThrottler(&retry.ThrottlerOptions{
        MaxTokens:  5,
        TokenRatio: 0.1,
    })
    
    user, err := retry.DoValue(ctx, func(ctx context.Context) (*User, error) {
        return getUserFromDatabase(ctx, "user123")
    }, 3)
    
    if err != nil {
        fmt.Printf("Database query failed: %v\n", err)
        return
    }
    
    fmt.Printf("Retrieved user: %+v\n", user)
}

type User struct {
    ID   string
    Name string
}

func getUserFromDatabase(ctx context.Context, userID string) (*User, error) {
    // Simulate database query
    return &User{ID: userID, Name: "John Doe"}, nil
}
```

### Microservice Communication

```go
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Configure for microservice calls
    r := retry.New().WithBackOff(
        retry.NewExponentialBackOff(100*time.Millisecond, 10*time.Second),
    )
    
    response, err := retry.DoValue(ctx, func(ctx context.Context) (*ServiceResponse, error) {
        return callMicroservice(ctx, "process-order", map[string]interface{}{
            "order_id": "12345",
            "amount":   99.99,
        })
    }, 5)
    
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("Service call timed out")
        } else if errors.Is(err, retry.ErrLimitExceeded) {
            fmt.Println("Exceeded maximum retry attempts")
        } else {
            fmt.Printf("Service call failed: %v\n", err)
        }
        return
    }
    
    fmt.Printf("Service response: %+v\n", response)
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
    
    r := retry.New().WithBackOff(
        retry.NewLinearBackOff(200 * time.Millisecond),
    )
    
    var successCount, failureCount int
    
    for _, item := range items {
        err := r.Do(ctx, func(ctx context.Context) error {
            return processItem(item)
        }, 3)
        
        if err != nil {
            fmt.Printf("Failed to process %s: %v\n", item, err)
            failureCount++
        } else {
            fmt.Printf("Successfully processed %s\n", item)
            successCount++
        }
    }
    
    fmt.Printf("Batch processing complete: %d success, %d failures\n", 
               successCount, failureCount)
}

func processItem(item string) error {
    // Simulate item processing
    return nil
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

1. **Use Context Timeouts**: Always use context with appropriate timeouts
2. **Choose Appropriate Backoff**: Exponential for external services, constant for internal operations
3. **Configure Throttling**: Enable throttling for rate-limited APIs
4. **Monitor Metrics**: Track retry attempts and success rates
5. **Set Reasonable Limits**: Balance between reliability and performance
6. **Handle Specific Errors**: Differentiate between temporary and permanent failures

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
