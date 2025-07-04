# Circuit Breaker Package

A distributed circuit breaker implementation for Go using Redis for coordination across multiple application instances. This circuit breaker prevents cascading failures by monitoring error rates and temporarily stopping requests to failing services.

## Features

- **Distributed Coordination**: Uses Redis pub/sub for state synchronization across instances
- **Multiple States**: Closed, Open, Half-Open, Disabled, and Forced-Open states
- **Configurable Thresholds**: Customizable failure ratios, counts, and durations
- **Slow Call Detection**: Configurable penalties for slow operations
- **Heartbeat Monitoring**: Optional periodic health checks
- **Type Safety**: Well-defined error types and state management
- **High Performance**: Optimized for low-latency operation

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/circuitbreaker
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alextanhongpin/core/dsync/circuitbreaker"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    // Create circuit breaker
    cb, stop := circuitbreaker.New(client, "my-service")
    defer stop()

    ctx := context.Background()

    // Define your operation
    operation := func() error {
        // Your potentially failing operation here
        // e.g., HTTP call, database query, etc.
        return callExternalService()
    }

    // Execute with circuit breaker protection
    err := cb.Do(ctx, operation)
    if err != nil {
        switch err {
        case circuitbreaker.ErrUnavailable:
            log.Println("Circuit breaker is open - service unavailable")
        case circuitbreaker.ErrForcedOpen:
            log.Println("Circuit breaker is forced open")
        default:
            log.Printf("Operation failed: %v", err)
        }
    }
}

func callExternalService() error {
    // Simulate external service call
    time.Sleep(100 * time.Millisecond)
    return nil
}
```

### Advanced Configuration

```go
// Create circuit breaker with custom settings
cb, stop := circuitbreaker.New(client, "my-service")
defer stop()

// Configure thresholds and durations
cb.FailureThreshold = 5                    // Open after 5 failures
cb.FailureRatio = 0.6                      // Open when 60% of requests fail
cb.BreakDuration = 30 * time.Second        // Stay open for 30 seconds
cb.SamplingDuration = 60 * time.Second     // Sample window of 60 seconds
cb.SuccessThreshold = 3                    // Close after 3 successes in half-open

// Custom failure counting
cb.FailureCount = func(err error) int {
    if isTimeoutError(err) {
        return 3 // Weight timeout errors more heavily
    }
    return 1
}

// Custom slow call detection
cb.SlowCallCount = func(duration time.Duration) int {
    if duration > 5*time.Second {
        return 2 // Penalize very slow calls
    }
    return 0
}

// Enable heartbeat monitoring
cb.HeartbeatDuration = 10 * time.Second
```

## Circuit Breaker States

### 1. Closed (Normal Operation)
- All requests are allowed through
- Monitors failure rate and slow calls
- Transitions to Open when thresholds are exceeded

### 2. Open (Blocking Requests)
- All requests are immediately rejected with `ErrUnavailable`
- After break duration, transitions to Half-Open
- State is synchronized across all instances via Redis

### 3. Half-Open (Testing Recovery)
- Limited requests are allowed through to test service recovery
- On success: counts toward closing the circuit
- On failure: immediately opens the circuit again
- Transitions to Closed after sufficient successes

### 4. Disabled (Bypass Mode)
- All requests pass through without monitoring
- Useful for maintenance or debugging

### 5. Forced-Open (Administrative Control)
- Similar to Open but manually controlled
- Returns `ErrForcedOpen` error
- Must be manually reset

## Error Types

```go
switch err {
case circuitbreaker.ErrUnavailable:
    // Circuit is open, service is considered unavailable
    // Should implement fallback or return cached data
    
case circuitbreaker.ErrForcedOpen:
    // Circuit is manually forced open
    // Usually indicates maintenance mode
    
default:
    // Actual error from your operation
    // Handle according to your business logic
}
```

## Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `FailureThreshold` | 10 | Minimum failures before opening |
| `FailureRatio` | 0.5 | Failure rate threshold (0.0-1.0) |
| `BreakDuration` | 5s | How long to stay open |
| `SamplingDuration` | 10s | Time window for rate calculation |
| `SuccessThreshold` | 5 | Successes needed to close from half-open |
| `HeartbeatDuration` | 0 | Heartbeat interval (0 = disabled) |

## Monitoring and Observability

### Check Circuit State

```go
status := cb.Status()
switch status {
case circuitbreaker.Closed:
    log.Println("Circuit is closed (normal)")
case circuitbreaker.Open:
    log.Println("Circuit is open (blocking)")
case circuitbreaker.HalfOpen:
    log.Println("Circuit is half-open (testing)")
case circuitbreaker.Disabled:
    log.Println("Circuit is disabled")
case circuitbreaker.ForcedOpen:
    log.Println("Circuit is forced open")
}
```

### Metrics Integration

```go
// Example with Prometheus metrics
err := cb.Do(ctx, operation)

// Record metrics
circuitBreakerRequests.WithLabelValues(cb.Status().String()).Inc()

if err != nil {
    if err == circuitbreaker.ErrUnavailable {
        circuitBreakerBlocked.Inc()
    } else {
        circuitBreakerErrors.Inc()
    }
} else {
    circuitBreakerSuccess.Inc()
}
```

## Best Practices

### 1. Choose Appropriate Thresholds
```go
// For high-traffic services
cb.FailureThreshold = 50
cb.FailureRatio = 0.7
cb.SamplingDuration = 30 * time.Second

// For low-traffic services  
cb.FailureThreshold = 5
cb.FailureRatio = 0.5
cb.SamplingDuration = 60 * time.Second
```

### 2. Implement Fallbacks
```go
err := cb.Do(ctx, primaryOperation)
if err == circuitbreaker.ErrUnavailable {
    // Circuit is open, use fallback
    return fallbackOperation()
}
return err
```

### 3. Use Unique Channels
```go
// Per service instance
userServiceCB, _ := circuitbreaker.New(client, "user-service")
paymentServiceCB, _ := circuitbreaker.New(client, "payment-service")

// Per endpoint if needed
getUserCB, _ := circuitbreaker.New(client, "user-service:get")
createUserCB, _ := circuitbreaker.New(client, "user-service:create")
```

### 4. Handle Cleanup Properly
```go
func NewService(redisClient *redis.Client) *Service {
    cb, stop := circuitbreaker.New(redisClient, "my-service")
    
    return &Service{
        cb: cb,
        stop: stop,
    }
}

func (s *Service) Close() error {
    s.stop() // Always call stop to cleanup resources
    return nil
}
```

## Performance Characteristics

Based on benchmarks:
- **Success Path**: ~1-2µs overhead per operation
- **Failure Path**: ~2-3µs overhead per operation  
- **Open Circuit**: ~100ns overhead (fast rejection)
- **Memory**: Minimal allocation overhead
- **Throughput**: 500k+ operations/second

## Redis Requirements

- Redis 3.0+ (for pub/sub functionality)
- Persistent connection recommended
- Consider Redis clustering for high availability

## Thread Safety

All operations are thread-safe and designed for concurrent use:
- State transitions are protected by mutexes
- Redis operations are atomic
- Multiple instances coordinate via pub/sub

## Examples

### HTTP Client with Circuit Breaker

```go
type HTTPClient struct {
    client *http.Client
    cb     *circuitbreaker.CircuitBreaker
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    cbErr := c.cb.Do(context.Background(), func() error {
        resp, err = c.client.Get(url)
        return err
    })
    
    if cbErr != nil {
        return nil, cbErr
    }
    
    return resp, err
}
```

### Database Connection with Circuit Breaker

```go
type Database struct {
    db *sql.DB
    cb *circuitbreaker.CircuitBreaker
}

func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
    var rows *sql.Rows
    var err error
    
    cbErr := d.cb.Do(context.Background(), func() error {
        rows, err = d.db.Query(query, args...)
        return err
    })
    
    if cbErr != nil {
        return nil, cbErr
    }
    
    return rows, err
}
```

## License

MIT License - see the LICENSE file for details.
