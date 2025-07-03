# HTTP Chain Package

The HTTP Chain package provides a simple and elegant way to compose HTTP middleware in Go web applications, enabling clean and maintainable request handling pipelines.

## Features

- **Clean Middleware Composition**: Compose multiple middleware in a readable manner
- **Proper Execution Order**: Middleware is applied in the expected order (first middleware wraps all others)
- **Handler Support**: Works with both `http.Handler` and `http.HandlerFunc` types
- **Minimal Overhead**: Lightweight implementation with no external dependencies
- **Idiomatic Go**: Follows standard Go patterns for HTTP handlers and middleware

## Quick Start

```go
package main

import (
    "net/http"
    "log"
    
    "github.com/alextanhongpin/core/http/chain"
)

func main() {
    // Create handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, world!"))
    })
    
    // Apply middleware chain
    wrappedHandler := chain.Handler(
        handler,
        loggingMiddleware,
        authMiddleware,
        timeoutMiddleware,
    )
    
    http.ListenAndServe(":8080", wrappedHandler)
}

// Middleware examples
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Authentication logic
        next.ServeHTTP(w, r)
    })
}

func timeoutMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Timeout logic
        next.ServeHTTP(w, r)
    })
}
```

## API Reference

### Handler Middleware

#### `Handler(h http.Handler, mws ...Middleware) http.Handler`

Chains multiple middleware with an HTTP handler.

```go
handler := chain.Handler(
    myHandler,
    middleware1,
    middleware2,
    middleware3,
)
```

### Handler Function Middleware

#### `HandlerFunc(h http.HandlerFunc, mws ...MiddlewareFunc) http.HandlerFunc`

Chains multiple middleware with an HTTP handler function.

```go
handlerFunc := chain.HandlerFunc(
    myHandlerFunc,
    middleware1Func,
    middleware2Func,
    middleware3Func,
)
```

## Understanding Middleware Order

The chain package applies middleware in the order they appear in the argument list, meaning:

```go
handler := chain.Handler(
    myHandler,
    first,
    second,
    third,
)
```

Results in an execution order of:

1. `first` middleware
2. `second` middleware
3. `third` middleware
4. `myHandler` handler

This is achieved by applying the middleware in reverse order:

```go
for i := len(mws) - 1; i > -1; i-- {
    h = mws[i](h)
}
```

## Creating Middleware

### Handler-based Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        log.Printf("Response: %s %s - %v", r.Method, r.URL.Path, time.Since(start))
    })
}
```

### HandlerFunc-based Middleware

```go
func timeoutMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        r = r.WithContext(ctx)
        next(w, r)
    }
}
```

## Common Usage Patterns

### Combining with Other Packages

The chain package works seamlessly with other packages in the library:

```go
import (
    "github.com/alextanhongpin/core/http/chain"
    "github.com/alextanhongpin/core/http/auth"
    "github.com/alextanhongpin/core/http/requestid"
    "github.com/alextanhongpin/core/http/middleware"
)

handler := chain.Handler(
    myHandler,
    middleware.RateLimit(100, time.Minute, middleware.ByIP),
    auth.BearerHandler(bearerConfig),
    requestid.Handler("X-Request-Id", generateID),
)
```

### Route-Specific Middleware

Apply different middleware chains to different routes:

```go
mux := http.NewServeMux()

// Public endpoints with minimal middleware
publicHandler := chain.Handler(
    publicEndpoint,
    loggingMiddleware,
)
mux.Handle("/public", publicHandler)

// Protected endpoints with authentication
protectedHandler := chain.Handler(
    privateEndpoint,
    loggingMiddleware,
    authMiddleware,
)
mux.Handle("/private", protectedHandler)
```

### Middleware with Configuration

Create configurable middleware factories:

```go
func withTimeout(duration time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), duration)
            defer cancel()
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

handler := chain.Handler(
    myHandler,
    withTimeout(5 * time.Second),
    loggingMiddleware,
)
```

## Best Practices

1. **Order Matters**: Place middleware in logical order (logging → auth → rate limiting → business logic)
2. **Early Returns**: Middleware should return early when preconditions aren't met
3. **Context Usage**: Use request context to pass data between middleware
4. **Clean Separation**: Each middleware should have a single responsibility
5. **Error Handling**: Properly handle errors in each middleware layer
