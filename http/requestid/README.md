# HTTP Request ID Package

The HTTP Request ID package provides utilities for generating, storing, and retrieving unique request identifiers for HTTP requests, enabling distributed tracing and request correlation.

## Features

- **Automatic ID Generation**: Generate unique identifiers for each request
- **Header Integration**: Extract and preserve IDs from request headers
- **Context Storage**: Store and retrieve request IDs via request context
- **Customizable Generation**: Configurable ID generation functions
- **Chain Compatibility**: Works seamlessly with middleware chains

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/google/uuid"
    
    "github.com/alextanhongpin/core/http/requestid"
    "github.com/alextanhongpin/core/http/chain"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/resource", handler)
    
    // Create request ID middleware with UUID generator
    idGenerator := func() string {
        return uuid.New().String()
    }
    
    // Apply middleware
    handler := chain.Handler(
        mux,
        requestid.Handler("X-Request-Id", idGenerator),
    )
    
    http.ListenAndServe(":8080", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Get request ID from context
    requestID, exists := requestid.Context.Value(r.Context())
    if exists {
        w.Header().Set("X-Request-Id", requestID)
    }
    
    // Handle request...
}
```

## API Reference

### Middleware

#### `Handler(header string, generator func() string) func(http.Handler) http.Handler`

Creates middleware that adds a request ID to each request's context.

```go
idMiddleware := requestid.Handler("X-Request-Id", func() string {
    return uuid.New().String()
})

// Apply middleware
handler = idMiddleware(handler)
```

### Context Access

#### `Context`

A context key for storing and retrieving request IDs.

```go
// Get request ID from context
requestID, exists := requestid.Context.Value(r.Context())
```

### Header Handling

The middleware automatically handles both incoming and outgoing request IDs:

1. If the request contains the configured header, its value is used
2. Otherwise, the generator function is called to create a new ID
3. The ID is stored in the request context for later access

## Custom ID Generators

You can provide any function that returns a string as the ID generator:

```go
// UUID-based generator
handler = requestid.Handler("X-Request-Id", func() string {
    return uuid.New().String()
})

// Sequential ID generator
var counter int64
handler = requestid.Handler("X-Request-Id", func() string {
    id := atomic.AddInt64(&counter, 1)
    return fmt.Sprintf("req-%d", id)
})

// Time-based generator
handler = requestid.Handler("X-Request-Id", func() string {
    return fmt.Sprintf("req-%d", time.Now().UnixNano())
})
```

## Logging Integration

Request IDs are particularly useful when integrated with logging:

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID, _ := requestid.Context.Value(r.Context())
        
        // Log with request ID
        log.Printf("[%s] %s %s", requestID, r.Method, r.URL.Path)
        
        next.ServeHTTP(w, r)
    })
}
```

## Distributed Tracing

Request IDs enable tracing requests across services:

```go
func CallExternalService(ctx context.Context, url string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    // Add request ID to outgoing request if available
    if requestID, exists := requestid.Context.Value(ctx); exists {
        req.Header.Set("X-Request-Id", requestID)
    }
    
    return http.DefaultClient.Do(req)
}
```

## Best Practices

1. **Consistent Header Names**: Use standardized header names like `X-Request-Id`
2. **Unique Generators**: Ensure your ID generator creates globally unique values
3. **Propagation**: Pass request IDs to downstream services and include in logs
4. **Early Middleware**: Apply the request ID middleware early in your middleware chain
5. **Response Headers**: Include the request ID in response headers for client-side correlation

## Integration with Logging

```go
func main() {
    handler := chain.Handler(
        mux,
        requestid.Handler("X-Request-Id", uuid.NewString),
        loggingMiddleware,
    )
    
    http.ListenAndServe(":8080", handler)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Get request ID
        requestID, _ := requestid.Context.Value(r.Context())
        
        // Create recorder to capture response
        recorder := response.NewResponseWriterRecorder(w)
        
        next.ServeHTTP(recorder, r)
        
        // Log after request with timing and status
        duration := time.Since(start)
        log.Printf(
            "[%s] %s %s - %d %s",
            requestID,
            r.Method,
            r.URL.Path,
            recorder.StatusCode(),
            duration,
        )
    })
}
```
