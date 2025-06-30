# Rate Limiting Library

A comprehensive, thread-safe, high-performance collection of rate limiting algorithms for Go. This library provides multiple rate limiting strategies including GCRA, Fixed Window, Sliding Window, and multi-key variants, each designed for different use cases and performance requirements.

## Overview

Rate limiting is essential for controlling traffic flow, preventing abuse, and ensuring system stability. This library implements industry-standard algorithms with:

- **Multiple Algorithms** - Choose the best fit for your use case
- **Thread-Safe** - Safe for concurrent use across multiple goroutines  
- **Zero Dependencies** - Uses only Go standard library
- **Memory Efficient** - Optimized for minimal memory footprint
- **Edge Case Protection** - Comprehensive input validation and overflow protection
- **Testing Support** - Injectable time functions for deterministic testing

## Available Rate Limiters

| Algorithm | Use Case | Memory | Precision | Burst Control |
|-----------|----------|--------|-----------|---------------|
| **GCRA** | High-precision, smooth distribution | Low | Nanosecond | Excellent |
| **Fixed Window** | Simple counting, less memory | Very Low | Window-based | Poor |
| **Sliding Window** | Balanced smoothness and efficiency | Medium | Sub-window | Good |
| **Multi-Key Variants** | Per-user/API key limiting | Medium+ | Algorithm-dependent | Algorithm-dependent |

## Installation

```bash
go get github.com/alextanhongpin/core/sync/ratelimit
```

## Quick Start

### GCRA (Recommended for most use cases)
```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
    // Create a GCRA rate limiter: 5 requests per second with burst of 2
    rl, err := ratelimit.NewGCRA(5, time.Second, 2)
    if err != nil {
        panic(err)
    }
    
    if rl.Allow() {
        fmt.Println("Request allowed")
    } else {
        fmt.Printf("Request denied, retry at: %v\n", rl.RetryAt())
    }
}
```

### Fixed Window (Simple counting)
```go
// Create a fixed window rate limiter: 100 requests per minute
rl, err := ratelimit.NewFixedWindow(100, time.Minute)
if err != nil {
    panic(err)
}

if rl.Allow() {
    fmt.Printf("Remaining: %d\n", rl.Remaining())
}
```

### Sliding Window (Balanced smoothness)
```go
// Create a sliding window rate limiter: 50 requests per 30 seconds
rl, err := ratelimit.NewSlidingWindow(50, 30*time.Second)
if err != nil {
    panic(err)
}

if rl.AllowN(5) {
    fmt.Printf("5 requests allowed, remaining: %d\n", rl.Remaining())
}
```

### Multi-Key Rate Limiting (Per-user limits)
```go
// Create a multi-key GCRA rate limiter: 10 requests per minute per user
rl, err := ratelimit.NewMultiGCRA(10, time.Minute, 2)
if err != nil {
    panic(err)
}

userID := "user123"
if rl.Allow(userID) {
    fmt.Printf("Request allowed for user %s\n", userID)
}
```

## API Reference

### GCRA (Generic Cell Rate Algorithm)

#### Constructors

##### `NewGCRA(limit int, period time.Duration, burst int) (*GCRA, error)`
Creates a new GCRA rate limiter with validation.

**Parameters:**
- `limit` - Number of requests allowed per period (must be > 0)
- `period` - Time period for the rate limit (must be > 0)  
- `burst` - Additional requests allowed as burst (must be ≥ 0)

##### `MustNewGCRA(limit int, period time.Duration, burst int) *GCRA`
Creates a new GCRA rate limiter and panics on validation errors.

#### Methods
- `Allow() bool` - Check if a single request is allowed
- `AllowN(n int) bool` - Check if N requests are allowed
- `RetryAt() time.Time` - Get the earliest retry time

### Fixed Window

#### Constructors

##### `NewFixedWindow(limit int, period time.Duration) (*FixedWindow, error)`
Creates a new fixed window rate limiter with validation.

##### `MustNewFixedWindow(limit int, period time.Duration) *FixedWindow`
Creates a new fixed window rate limiter and panics on validation errors.

#### Methods
- `Allow() bool` - Check if a single request is allowed
- `AllowN(n int) bool` - Check if N requests are allowed
- `Remaining() int` - Get the number of remaining requests in current window
- `RetryAt() time.Time` - Get the earliest retry time

### Sliding Window

#### Constructors

##### `NewSlidingWindow(limit int, period time.Duration) (*SlidingWindow, error)`
Creates a new sliding window rate limiter with validation.

##### `MustNewSlidingWindow(limit int, period time.Duration) *SlidingWindow`
Creates a new sliding window rate limiter and panics on validation errors.

#### Methods
- `Allow() bool` - Check if a single request is allowed
- `AllowN(n int) bool` - Check if N requests are allowed
- `Remaining() int` - Get the approximate number of remaining requests

### Multi-Key Variants

All multi-key rate limiters have similar APIs but require a `key` parameter:

#### Multi Fixed Window
```go
rl, _ := ratelimit.NewMultiFixedWindow(100, time.Hour)
rl.Allow("user123")           // Check single request for user
rl.AllowN("user123", 5)       // Check 5 requests for user
```

#### Multi Sliding Window  
```go
rl, _ := ratelimit.NewMultiSlidingWindow(50, 30*time.Second)
rl.Allow("api_key_456")       // Check single request for API key
```

#### Multi GCRA
```go
rl, _ := ratelimit.NewMultiGCRA(10, time.Minute, 2)
rl.Allow("session_789")       // Check single request for session
rl.RetryAt("session_789")     // Get retry time for session
```

## Algorithm Details

### GCRA (Generic Cell Rate Algorithm)
GCRA tracks the theoretical "virtual scheduling time" for requests. Each request advances this time by the interval between requests (period/limit). A request is allowed if the current virtual time minus the burst allowance is not greater than the current real time.

**Key Concepts:**
- **Interval**: Time between consecutive requests = `period / limit`
- **Offset**: Burst allowance in time units = `interval × burst`  
- **Virtual Time**: Tracks when requests would be scheduled in ideal conditions

**Best for**: High-precision applications, APIs requiring smooth traffic distribution

### Fixed Window
Counts requests within fixed time windows. Simple but can allow traffic bursts at window boundaries.

**How it works:**
1. Reset counter at the start of each time window
2. Increment counter for each request
3. Reject requests when counter exceeds limit

**Best for**: Simple rate limiting, memory-constrained environments

### Sliding Window  
Uses a combination of current and previous window counts with time-based interpolation to provide smoother rate limiting than fixed windows.

**How it works:**
1. Maintain counters for current and previous windows
2. Calculate weighted average based on time position within current window
3. More accurate than fixed window, less precise than GCRA

**Best for**: Balanced smoothness and memory efficiency

### Multi-Key Variants
All algorithms have multi-key variants that maintain separate state per key (user ID, API key, etc.). Useful for per-user or per-tenant rate limiting.

**Memory considerations**: Memory usage scales with the number of unique keys

## Configuration Examples

### API Rate Limiting
```go
// Public API: 1000 requests per hour with 50 request burst
publicAPI, _ := ratelimit.NewGCRA(1000, time.Hour, 50)

// Premium API: 10,000 requests per hour with 200 request burst  
premiumAPI, _ := ratelimit.NewGCRA(10000, time.Hour, 200)

// Per-user limiting: 100 requests per 15 minutes per user
perUser, _ := ratelimit.NewMultiGCRA(100, 15*time.Minute, 10)
```

### Microservice Protection
```go
// Database connection limiting: 50 queries per second
dbQueries, _ := ratelimit.NewFixedWindow(50, time.Second)

// External API calls: 100 calls per minute with sliding window
externalAPI, _ := ratelimit.NewSlidingWindow(100, time.Minute)

// Memory-sensitive service: Fixed window for minimal overhead
lightweight, _ := ratelimit.NewFixedWindow(1000, time.Second)
```

### High-Frequency Operations
```go
// Real-time metrics: 10,000 requests per second
metrics, _ := ratelimit.NewGCRA(10000, time.Second, 100)

// WebSocket connections: per-connection limiting
wsConnections, _ := ratelimit.NewMultiSlidingWindow(50, 10*time.Second)
```

### Conservative Rate Limiting
```go
// Critical operations: 1 request per 5 seconds, no burst
critical, _ := ratelimit.NewGCRA(1, 5*time.Second, 0)

// Admin operations: 10 requests per minute with fixed window
admin, _ := ratelimit.NewFixedWindow(10, time.Minute)
```

## Error Handling

All rate limiters provide comprehensive error validation:

```go
// GCRA errors
rl, err := ratelimit.NewGCRA(-1, time.Second, 0)
if err != nil {
    switch err {
    case ratelimit.ErrInvalidLimit:
        // Handle invalid limit
    case ratelimit.ErrInvalidPeriod:
        // Handle invalid period
    case ratelimit.ErrInvalidBurst:
        // Handle invalid burst
    }
}

// Fixed Window errors
fw, err := ratelimit.NewFixedWindow(0, time.Second)
if err != nil {
    switch err {
    case ratelimit.ErrInvalidFixedWindowLimit:
        // Handle invalid limit
    case ratelimit.ErrInvalidFixedWindowPeriod:
        // Handle invalid period
    }
}

// Similar patterns for other rate limiters...
```

### Error Types

**GCRA:**
- `ErrInvalidLimit` - Limit must be positive
- `ErrInvalidPeriod` - Period must be positive
- `ErrInvalidBurst` - Burst cannot be negative

**Fixed Window:**
- `ErrInvalidFixedWindowLimit` - Limit must be positive
- `ErrInvalidFixedWindowPeriod` - Period must be positive

**Sliding Window:**
- `ErrInvalidSlidingWindowLimit` - Limit must be positive
- `ErrInvalidSlidingWindowPeriod` - Period must be positive

**Multi-Key Variants:**
- Similar to single-key versions with prefix like `ErrInvalidMultiGCRALimit`
- Additional validation for empty keys in `AllowN` methods

## Testing Support

All rate limiters support dependency injection for time, enabling deterministic testing:

```go
func TestRateLimit(t *testing.T) {
    rl := ratelimit.MustNewGCRA(2, time.Second, 0)
    
    now := time.Now()
    rl.Now = func() time.Time { return now }
    
    // First request should be allowed
    assert.True(t, rl.Allow())
    
    // Advance time by 500ms
    rl.Now = func() time.Time { return now.Add(500 * time.Millisecond) }
    
    // Second request should be allowed (500ms = 1/2 second interval)
    assert.True(t, rl.Allow())
    
    // Third request should be denied (not enough time passed)
    assert.False(t, rl.Allow())
}

func TestMultiKeyRateLimit(t *testing.T) {
    rl := ratelimit.MustNewMultiFixedWindow(3, time.Second)
    
    now := time.Now()
    rl.Now = func() time.Time { return now }
    
    // Different users should have independent limits
    assert.True(t, rl.Allow("user1"))
    assert.True(t, rl.Allow("user2"))
    assert.True(t, rl.Allow("user1"))
}
```

## Performance Characteristics

| Algorithm | Time Complexity | Space Complexity | Memory Usage | Thread Safety |
|-----------|-----------------|------------------|--------------|---------------|
| **GCRA** | O(1) | O(1) | ~64 bytes | Read-write mutex |
| **Fixed Window** | O(1) | O(1) | ~48 bytes | Read-write mutex |
| **Sliding Window** | O(1) | O(1) | ~56 bytes | Read-write mutex |
| **Multi Fixed Window** | O(1) | O(k) | ~48 bytes + 32*k | Read-write mutex |
| **Multi Sliding Window** | O(1) | O(k) | ~56 bytes + 40*k | Read-write mutex |
| **Multi GCRA** | O(1) | O(k) | ~64 bytes + 24*k | Read-write mutex |

*k = number of unique keys*

## Choosing the Right Algorithm

### Decision Matrix

| Use Case | Recommended Algorithm | Reason |
|----------|----------------------|---------|
| **High-precision APIs** | GCRA | Smooth distribution, nanosecond precision |
| **Simple rate limiting** | Fixed Window | Minimal memory, easy to understand |
| **Balanced performance** | Sliding Window | Good smoothness vs efficiency trade-off |
| **Per-user/tenant limits** | Multi-* variants | Isolated limits per key |
| **Memory-constrained** | Fixed Window | Lowest memory footprint |
| **Burst-sensitive** | GCRA | Excellent burst control |
| **WebSocket/real-time** | GCRA or Sliding Window | Smooth traffic handling |

### Performance Guidelines

- **GCRA**: Best overall choice for most applications
- **Fixed Window**: Use when memory is extremely limited
- **Sliding Window**: Good middle ground for moderate precision needs
- **Multi-Key**: Add 24-40 bytes per unique key, use memory cleanup strategies

### Algorithm Comparison

| Algorithm | Smoothness | Burst Control | Memory | Precision | Complexity |
|-----------|------------|---------------|--------|-----------|------------|
| GCRA | Excellent | Excellent | Low | Nanosecond | Medium |
| Fixed Window | Poor | Poor | Very Low | Window-based | Simple |
| Sliding Window | Good | Fair | Medium | Sub-window | Medium |
| Token Bucket | Good | Good | Low | Configurable | Medium |

## Edge Cases Handled

All rate limiter implementations include protection against:

1. **Integer Overflow** - Prevents overflow in timestamp calculations
2. **Invalid Parameters** - Comprehensive input validation with specific error types
3. **Extreme Values** - Handles very large burst values and periods gracefully
4. **Concurrent Access** - Thread-safe with proper read-write locking
5. **Time Precision** - Nanosecond precision with overflow protection
6. **Empty Keys** - Multi-key variants validate against empty key strings
7. **Negative Values** - All parameters validated for appropriate ranges
8. **Zero Division** - Protected against division by zero in rate calculations

### Comprehensive Testing

The library includes extensive edge case testing:
- Input validation for all constructors
- Concurrent access patterns
- Integer overflow scenarios  
- Extreme parameter values
- High-frequency operations
- Large burst configurations

## Best Practices

### Choosing Parameters

1. **Start Conservative** - Begin with lower limits and increase based on monitoring
2. **Monitor Metrics** - Track rejection rates, retry patterns, and system performance
3. **Consider Burst** - Set burst based on expected traffic patterns and system capacity
4. **Test Thoroughly** - Validate rate limiting behavior under realistic load conditions
5. **Use Multi-Key Wisely** - Consider memory implications for large key spaces

### Production Usage

```go
// Production-ready HTTP middleware
func NewAPIRateLimiter() (*ratelimit.GCRA, error) {
    return ratelimit.NewGCRA(
        1000,              // 1000 requests
        time.Hour,         // per hour
        50,                // with 50 request burst
    )
}

func RateLimitMiddleware(rl *ratelimit.GCRA) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !rl.Allow() {
                retryAfter := rl.RetryAt().Sub(time.Now())
                w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.WriteHeader(http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Per-user rate limiting with cleanup
type UserRateLimiter struct {
    limiter   *ratelimit.MultiGCRA
    mu        sync.RWMutex
    lastSeen  map[string]time.Time
    cleanupInterval time.Duration
}

func NewUserRateLimiter(limit int, period time.Duration, burst int) *UserRateLimiter {
    rl := &UserRateLimiter{
        limiter:   ratelimit.MustNewMultiGCRA(limit, period, burst),
        lastSeen:  make(map[string]time.Time),
        cleanupInterval: time.Hour,
    }
    
    // Start cleanup goroutine
    go rl.cleanup()
    return rl
}

func (rl *UserRateLimiter) Allow(userID string) bool {
    rl.mu.Lock()
    rl.lastSeen[userID] = time.Now()
    rl.mu.Unlock()
    
    return rl.limiter.Allow(userID)
}

func (rl *UserRateLimiter) cleanup() {
    ticker := time.NewTicker(rl.cleanupInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        rl.mu.Lock()
        cutoff := time.Now().Add(-24 * time.Hour)
        for userID, lastSeen := range rl.lastSeen {
            if lastSeen.Before(cutoff) {
                delete(rl.lastSeen, userID)
                // Note: MultiGCRA doesn't expose state cleanup,
                // consider implementing if memory is critical
            }
        }
        rl.mu.Unlock()
    }
}
```

### Memory Management for Multi-Key Limiters

```go
// For applications with many keys, implement periodic cleanup
type CleanupConfig struct {
    MaxAge      time.Duration
    CleanupInterval time.Duration
    MaxKeys     int
}

func (config CleanupConfig) ShouldCleanup(keyCount int, lastCleanup time.Time) bool {
    return keyCount > config.MaxKeys || 
           time.Since(lastCleanup) > config.CleanupInterval
}
```

## License

This implementation is part of the [alextanhongpin/core](https://github.com/alextanhongpin/core) library and is available under the MIT License.

## References

- [ATM Forum Traffic Management Specification](https://www.broadband-forum.org/technical/download/af-tm-0121.000.pdf) - Original GCRA specification
- [Generic Cell Rate Algorithm on Wikipedia](https://en.wikipedia.org/wiki/Generic_cell_rate_algorithm) - Algorithm overview
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket) - Alternative approach comparison
- [Sliding Window Rate Limiting](https://konghq.com/blog/how-to-design-a-scalable-rate-limiting-algorithm) - Algorithm comparison and analysis
- [Traffic Shaping and Rate Limiting Techniques](https://tools.ietf.org/html/rfc2475) - IETF standards for traffic control
