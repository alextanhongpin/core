# Rate Limiting Package

A high-performance, Redis-based rate limiting library for Go, implementing multiple rate limiting algorithms for different use cases.

## Features

- **Fixed Window**: Simple, burst-tolerant rate limiting with fixed time windows
- **GCRA (Generic Cell Rate Algorithm)**: Smooth rate limiting with configurable burst capacity
- **Redis-based**: Distributed rate limiting across multiple application instances
- **Atomic Operations**: Uses Redis Lua scripts for race-condition-free operations
- **High Performance**: Minimal Redis round trips with optimized scripts
- **Flexible API**: Support for single and bulk token consumption

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/ratelimit
```

## Quick Start

### Fixed Window Rate Limiting

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/alextanhongpin/core/dsync/ratelimit"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()
    
    // Create fixed window rate limiter: 10 requests per minute
    rl := ratelimit.NewFixedWindow(client, 10, time.Minute)
    
    ctx := context.Background()
    userKey := "user:123"
    
    // Perform a single check and receive a detailed Result.
    result, err := rl.Limit(ctx, userKey)
    if err != nil {
        log.Fatal(err)
    }

    if result.Allow {
        log.Println("Request allowed")
    } else {
        log.Println("Rate limit exceeded")
    }

    log.Printf("Remaining requests: %d", result.Remaining)
    log.Printf("Resets in: %v", result.ResetAfter)
}
```

### GCRA Rate Limiting

```go
// Create GCRA rate limiter: 100 requests per second with burst of 10
rl := ratelimit.NewGCRA(client, 100, time.Second, 10)

ctx := context.Background()
apiKey := "api:abc123"

// Single request
allowed, err := rl.Allow(ctx, apiKey)
if err != nil {
    log.Fatal(err)
}

// Bulk requests (e.g., batch processing)
allowed, err = rl.AllowN(ctx, apiKey, 5)
if err != nil {
    log.Fatal(err)
}
```

## Algorithms

### Fixed Window

The Fixed Window algorithm divides time into fixed intervals and allows a specified number of requests per interval.

**Characteristics:**
- Simple to understand and implement
- Allows bursts at window boundaries
- Memory efficient
- Predictable reset times

**Use Cases:**
- API rate limiting with simple quotas
- Resource protection with burst tolerance
- User-facing rate limits with clear reset times

**Example**: 1000 requests per hour
- Window 1 (00:00-01:00): Up to 1000 requests allowed
- Window 2 (01:00-02:00): Counter resets, up to 1000 requests allowed

### GCRA (Generic Cell Rate Algorithm)

GCRA provides smooth rate limiting by tracking the theoretical arrival time of the next request.

**Characteristics:**
- Smooth rate limiting (no boundary bursts)
- Configurable burst capacity
- More complex but fairer distribution
- Better for sustained traffic patterns

**Use Cases:**
- High-throughput APIs requiring smooth traffic
- Systems sensitive to traffic spikes
- Services with strict SLA requirements

**Parameters:**
- `limit`: Maximum requests per period
- `period`: Time window duration
- `burst`: Additional burst capacity (0 = no burst)

## API Reference

### Fixed Window

```go
type FixedWindow struct {}

// Create new fixed window rate limiter
func NewFixedWindow(client *redis.Client, limit int, period time.Duration) *FixedWindow

// Check if single request is allowed
func (r *FixedWindow) Allow(ctx context.Context, key string) (bool, error)

// Check if N requests are allowed
func (r *FixedWindow) AllowN(ctx context.Context, key string, n int) (bool, error)

// Get remaining requests in current window
func (r *FixedWindow) Limit(ctx context.Context, key string) (*Result, error)

func (r *FixedWindow) LimitN(ctx context.Context, key string, n int) (*Result, error)
```

### GCRA

```go
type GCRA struct {}

// Create new GCRA rate limiter
func NewGCRA(client *redis.Client, limit int, period time.Duration, burst int) *GCRA

// Check if a single request is allowed
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error)

// Check if N requests are allowed
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error)

// Get detailed result for a single request
func (g *GCRA) Limit(ctx context.Context, key string) (*Result, error)

// Get detailed result for N requests
func (g *GCRA) LimitN(ctx context.Context, key string, n int) (*Result, error)
```

## How to Use the Package

The package exposes two main limiter types:

* **Fixed Window** – a simple counter that resets every *period*.
* **GCRA** – a smooth rate‑limiter based on the Generic Cell Rate
  Algorithm.

Both are constructed with a Redis client and configured with the
desired limits.  Once created, you can call `Allow`, `AllowN`,
`Remaining` and `ResetAfter` on any key.  Keys are typically
derived from user IDs, API keys, or any string that uniquely
identifies a client.

```go
// Common pattern
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer client.Close()

// Fixed window – 100 requests per hour
fixed := ratelimit.NewFixedWindow(client, 100, time.Hour)

// GCRA – 50 requests per second with burst of 20
gcra := ratelimit.NewGCRA(client, 50, time.Second, 20)
```

Now you can call the rate‑limit methods in your handlers.

## Usage Patterns

### Per-User Rate Limiting

```go
func handleAPIRequest(userID string) error {
    rl := ratelimit.NewFixedWindow(redisClient, 1000, time.Hour)
    
    allowed, err := rl.Allow(ctx, fmt.Sprintf("user:%s", userID))
    if err != nil {
        return err
    }
    
    if !allowed {
        return errors.New("rate limit exceeded")
    }
    
    // Process request
    return nil
}
```

### API Key Rate Limiting

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
    rl := ratelimit.NewGCRA(redisClient, 100, time.Minute, 10)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")

        allowed, err := rl.Allow(r.Context(), fmt.Sprintf("api:%s", apiKey))
        if err != nil {
            http.Error(w, "Internal error", http.StatusInternalServerError)
            return
        }

        if !allowed {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

## Real‑World Example
The following demonstrates a common pattern: protecting an API endpoint that
creates resources for a user.  We limit each user to **10 writes per minute**.

```go
func createResourceHandler(db *sql.DB, rl *ratelimit.FixedWindow) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := r.Header.Get("X-User-ID")
        if userID == "" {
            http.Error(w, "Missing user ID", http.StatusBadRequest)
            return
        }

        allowed, err := rl.Allow(r.Context(), fmt.Sprintf("user:%s", userID))
        if err != nil {
            http.Error(w, "Internal error", http.StatusInternalServerError)
            return
        }
        if !allowed {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        // …create resource in db…
        w.WriteHeader(http.StatusCreated)
    }
}
```

### Bulk Operations

```go
func processBatch(items []Item) error {
    rl := ratelimit.NewGCRA(redisClient, 1000, time.Second, 50)
    
    batchSize := len(items)
    allowed, err := rl.AllowN(ctx, "batch:processor", batchSize)
    if err != nil {
        return err
    }
    
    if !allowed {
        return errors.New("insufficient rate limit capacity")
    }
    
    // Process all items
    return processBatchItems(items)
}
```

### Rate Limiting with Graceful Degradation

```go
func handleRequest(priority string) error {
    var rl RateLimiter
    
    switch priority {
    case "premium":
        rl = ratelimit.NewGCRA(redisClient, 1000, time.Minute, 100)
    case "standard":
        rl = ratelimit.NewFixedWindow(redisClient, 100, time.Minute)
    default:
        rl = ratelimit.NewFixedWindow(redisClient, 10, time.Minute)
    }
    
    allowed, err := rl.Allow(ctx, fmt.Sprintf("user:%s", userID))
    if err != nil {
        // Fail open on Redis errors
        log.Printf("Rate limit check failed: %v", err)
        return nil
    }
    
    if !allowed {
        return errors.New("rate limit exceeded")
    }
    
    return processRequest()
}
```

## Performance Characteristics

### Fixed Window
- **Time Complexity**: O(1)
- **Space Complexity**: O(1) per key
- **Redis Operations**: 1-2 per request
- **Memory Usage**: ~32 bytes per key
- **Throughput**: High (limited by Redis)

### GCRA
- **Time Complexity**: O(1)
- **Space Complexity**: O(1) per key
- **Redis Operations**: 1-2 per request
- **Memory Usage**: ~32 bytes per key
- **Throughput**: High (limited by Redis)

## Best Practices

### Key Naming

Use hierarchical, descriptive keys:
```go
// Good
userKey := fmt.Sprintf("user:%s:api", userID)
ipKey := fmt.Sprintf("ip:%s:login", clientIP)

// Avoid
key := userID  // Too generic
key := "user_" + userID + "_api_limit"  // Hard to parse
```

### Error Handling

```go
allowed, err := rl.Allow(ctx, key)
if err != nil {
    // Log error but don't block request
    log.Printf("Rate limit check failed: %v", err)
    // Implement fallback strategy:
    // - Fail open (allow request)
    // - Fail closed (deny request) 
    // - Use local rate limiting
    return true, nil  // Fail open
}
```

### Redis Connection Management

```go
// Use connection pooling
client := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     20,
    MinIdleConns: 5,
    PoolTimeout:  30 * time.Second,
})

// Monitor Redis health
go func() {
    for {
        err := client.Ping(context.Background()).Err()
        if err != nil {
            log.Printf("Redis health check failed: %v", err)
        }
        time.Sleep(30 * time.Second)
    }
}()
```

### Monitoring and Observability

```go
// Track rate limiting metrics
type rateLimitMetrics struct {
    allowed   prometheus.Counter
    denied    prometheus.Counter
    errors    prometheus.Counter
    latency   prometheus.Histogram
}

func (m *rateLimitMetrics) trackRequest(allowed bool, err error, duration time.Duration) {
    m.latency.Observe(duration.Seconds())
    
    if err != nil {
        m.errors.Inc()
        return
    }
    
    if allowed {
        m.allowed.Inc()
    } else {
        m.denied.Inc()
    }
}
```

## Metrics & Observability

Both `FixedWindow` and `GCRA` support pluggable metrics collectors for tracking requests, allowed, and denied counts. You can use the built-in atomic collector for in-memory stats, or integrate with Prometheus for production monitoring.

### Using the Atomic Metrics Collector (default)

By default, if you do not provide a metrics collector, an atomic in-memory collector is used:

```go
rl := ratelimit.NewFixedWindow(client, 10, time.Minute) // uses AtomicMetricsCollector by default
```

### Using Prometheus for Metrics

To collect metrics with Prometheus, inject a `PrometheusMetricsCollector`:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/alextanhongpin/core/dsync/ratelimit"
)

totalRequests := prometheus.NewCounter(prometheus.CounterOpts{Name: "fw_total_requests", Help: "Total FixedWindow requests."})
allowed := prometheus.NewCounter(prometheus.CounterOpts{Name: "fw_allowed", Help: "Allowed requests."})
denied := prometheus.NewCounter(prometheus.CounterOpts{Name: "fw_denied", Help: "Denied requests."})
prometheus.MustRegister(totalRequests, allowed, denied)

metrics := &ratelimit.PrometheusMetricsCollector{
    TotalRequests: totalRequests,
    Allowed:       allowed,
    Denied:        denied,
}
rl := ratelimit.NewFixedWindow(client, 10, time.Minute, metrics)
```

The same pattern applies to `GCRA`:

```go
metrics := &ratelimit.PrometheusMetricsCollector{
    TotalRequests: prometheus.NewCounter(...),
    Allowed:       prometheus.NewCounter(...),
    Denied:        prometheus.NewCounter(...),
}
rl := ratelimit.NewGCRA(client, 100, time.Second, 10, metrics)
```

See the GoDoc for the `MetricsCollector` interface and available implementations.

## Configuration Examples

### High-Throughput API

```go
// For APIs handling thousands of requests per second
rl := ratelimit.NewGCRA(client, 10000, time.Second, 1000)
```

### User-Facing Application

```go
// For web applications with human users
rl := ratelimit.NewFixedWindow(client, 60, time.Minute)
```

### Background Job Processing

```go
// For batch processing systems
rl := ratelimit.NewGCRA(client, 100, time.Second, 0)  // No burst
```

## Limitations

1. **Redis Dependency**: Requires Redis for distributed operation
2. **Network Latency**: Each check requires network round trip
3. **Single Algorithm per Instance**: Cannot mix algorithms for same key
4. **No Built-in Persistence**: Rate limit state lost on Redis restart
5. **Clock Skew Sensitivity**: GCRA sensitive to system clock differences

## Contributing

1. Add tests for new features
2. Update documentation
3. Run benchmarks for performance changes
4. Follow existing code patterns
5. Ensure backward compatibility

## License

MIT License
