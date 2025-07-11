# HTTP Middleware Package

The HTTP Middleware package provides a collection of middleware components for Go web applications, including rate limiting, logging, and security features.

## Features

- **Rate Limiting**: Token bucket algorithm for limiting request rates
- **Per-Key Limits**: Rate limit by IP, API key, user ID, or custom values
- **Memory Efficient**: Automatic cleanup of stale rate limiting data
- **Configurable**: Adjustable request rates, time windows, and key functions
- **Production Ready**: Thread-safe implementation with proper locking
- **Cross-Package Integration**: Works with `chain`, `auth`, and `requestid` middleware

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
    // Apply rate limiting middleware (60 requests/minute per IP)
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

## Best Practices

- Use per-key rate limiting for user-based or API key-based quotas.
- Integrate with authentication and request ID middleware for full request context.
- Monitor and tune rate limits for production workloads.

## Related Packages

- [`chain`](../chain/README.md): Middleware chaining
- [`auth`](../auth/README.md): Authentication middleware
- [`requestid`](../requestid/README.md): Request ID propagation

## License

MIT
