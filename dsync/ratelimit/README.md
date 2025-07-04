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
    
    // Check if request is allowed
    allowed, err := rl.Allow(ctx, userKey)
    if err != nil {
        log.Fatal(err)
    }
    
    if allowed {
        log.Println("Request allowed")
    } else {
        log.Println("Rate limit exceeded")
    }
    
    // Check remaining capacity
    remaining, err := rl.Remaining(ctx, userKey)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Remaining requests: %d", remaining)
    
    // Check when limit resets
    resetAfter, err := rl.ResetAfter(ctx, userKey)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Resets in: %v", resetAfter)
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
func (r *FixedWindow) Remaining(ctx context.Context, key string) (int, error)

// Get time until window resets
func (r *FixedWindow) ResetAfter(ctx context.Context, key string) (time.Duration, error)
```

### GCRA

```go
type GCRA struct {}

// Create new GCRA rate limiter
func NewGCRA(client *redis.Client, limit int, period time.Duration, burst int) *GCRA

// Check if single request is allowed
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error)

// Check if N requests are allowed
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error)
```

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
            http.Error(w, "Internal error", 500)
            return
        }
        
        if !allowed {
            http.Error(w, "Rate limit exceeded", 429)
            return
        }
        
        next.ServeHTTP(w, r)
    })
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
