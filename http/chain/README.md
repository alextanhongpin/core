# HTTP Chain Package

The HTTP Chain package provides a simple, idiomatic way to compose HTTP middleware in Go web applications. It enables clean, maintainable request handling pipelines and works seamlessly with other core packages.

## Features
- Clean, readable middleware composition
- Proper execution order (first middleware wraps all others)
- Support for both `http.Handler` and `http.HandlerFunc`
- Minimal overhead, no external dependencies
- Idiomatic Go patterns for HTTP middleware

## Quick Start
```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/chain"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, world!"))
    })

    // Compose middleware chain
    wrapped := chain.Handler(
        handler,
        loggingMiddleware,
        authMiddleware,
        rateLimitMiddleware,
    )

    http.ListenAndServe(":8080", wrapped)
}

// Example middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

## API Reference
- `Handler(h http.Handler, mws ...Middleware) http.Handler` — Compose middleware for handlers
- `HandlerFunc(h http.HandlerFunc, mws ...MiddlewareFunc) http.HandlerFunc` — Compose for handler functions

## Best Practices
- Order middleware from outermost (logging, tracing) to innermost (auth, validation)
- Use with other core packages for robust pipelines

## See Also
- [`auth`](../auth/README.md) — Authentication middleware
- [`middleware`](../middleware/README.md) — Rate limiting, logging, security
