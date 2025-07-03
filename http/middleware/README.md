# HTTP Middleware Package

The HTTP Middleware package provides a collection of middleware components for Go web applications, including rate limiting, logging, and security features.

## Features

- **Rate Limiting**: Token bucket algorithm for limiting request rates
- **Per-Key Limits**: Rate limit by IP, API key, user ID, or custom values
- **Memory Efficient**: Automatic cleanup of stale rate limiting data
- **Configurable**: Adjustable request rates, time windows, and key functions
- **Production Ready**: Thread-safe implementation with proper locking

## Quick Start

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/http/middleware"
    "github.com/alextanhongpin/core/http/chain"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/resource", apiHandler)
    
    // Apply rate limiting middleware
    // Allow 60 requests per minute per IP address
    handler := chain.Handler(
        mux,
        middleware.RateLimit(60, time.Minute, middleware.ByIP),
    )
    
    http.ListenAndServe(":8080", handler)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("API response"))
}
```

## API Reference

### Rate Limiting

#### `RateLimit(requests int, window time.Duration, keyFunc KeyFunc) func(http.Handler) http.Handler`

Creates middleware that limits request rates using the token bucket algorithm.

```go
// Allow 100 requests per minute per IP
handler = middleware.RateLimit(100, time.Minute, middleware.ByIP)(handler)

// Allow 1000 requests per hour per API key
handler = middleware.RateLimit(1000, time.Hour, middleware.ByAPIKey)(handler)
```

### Key Functions

#### `ByIP(r *http.Request) string`

Extracts the client's IP address as the rate limiting key.

```go
handler = middleware.RateLimit(60, time.Minute, middleware.ByIP)(handler)
```

#### `ByAPIKey(r *http.Request) string`

Uses the Authorization header as the rate limiting key.

```go
handler = middleware.RateLimit(100, time.Minute, middleware.ByAPIKey)(handler)
```

#### `ByUserID(userContextKey string) KeyFunc`

Creates a key function that extracts the user ID from request context.

```go
// Assuming authentication middleware sets user ID in context
handler = middleware.RateLimit(100, time.Minute, 
    middleware.ByUserID("user_id"),
)(handler)
```

### Custom Key Functions

You can create custom key functions for specialized rate limiting:

```go
// Rate limit by path and IP together
pathAndIPKey := func(r *http.Request) string {
    ip := middleware.ByIP(r)
    return fmt.Sprintf("%s:%s", r.URL.Path, ip)
}

handler = middleware.RateLimit(10, time.Minute, pathAndIPKey)(handler)
```

## Advanced Usage

### Rate Limiter Configuration

```go
// Create a custom rate limiter
limiter := middleware.NewRateLimiter(100, time.Minute)

// Create middleware with custom limiter
customLimit := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := getCustomKey(r)
        
        if !limiter.Allow(key) {
            w.WriteHeader(http.StatusTooManyRequests)
            w.Write([]byte("Rate limit exceeded"))
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Multiple Rate Limiters

Apply different rate limits for different resources or user types:

```go
// General API limit
apiLimiter := middleware.RateLimit(1000, time.Hour, middleware.ByIP)

// Stricter limit for authentication endpoints
authLimiter := middleware.RateLimit(10, time.Minute, middleware.ByIP)

mux.Handle("/api/", apiLimiter(apiHandler))
mux.Handle("/auth/", authLimiter(authHandler))
```

### Combined with Authentication

Rate limiting works well with authentication middleware:

```go
// Authentication middleware sets user in context
authMiddleware := auth.RequireBearerHandler

// Apply middlewares in order
handler := chain.Handler(
    apiHandler,
    middleware.RateLimit(100, time.Minute, middleware.ByUserID("user")),
    authMiddleware,
)
```

## The Token Bucket Algorithm

The rate limiter uses a token bucket algorithm:

1. Each client gets a bucket with a maximum number of tokens
2. Each request consumes one token
3. Tokens refill at a specified rate over time
4. When a bucket is empty, requests are rejected
5. Stale buckets are automatically cleaned up

```go
// Allow 60 requests initially and refill at 1 request per second
handler = middleware.RateLimit(60, time.Minute, middleware.ByIP)(handler)
```

## Custom Response Headers

The rate limiting middleware adds informative headers:

- `X-RateLimit-Limit`: Total allowed requests in the window
- `X-RateLimit-Window`: The time window for the limit

## Best Practices

1. **Start Liberal**: Begin with generous limits and tighten as needed
2. **Layered Limits**: Apply different limits to different API endpoints
3. **Response Headers**: Include rate limit information in responses
4. **Scaling**: For distributed systems, consider Redis-based rate limiters
5. **Monitoring**: Track rate limit rejections to identify abuse

## Performance Considerations

The in-memory rate limiter is designed for single-instance applications:

- Uses minimal memory with automatic cleanup
- Thread-safe with proper mutex locking
- Efficient token calculation without storing timestamps for each request

For distributed systems, consider using a shared cache like Redis for rate limiting.
