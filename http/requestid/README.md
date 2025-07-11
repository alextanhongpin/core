# HTTP Request ID Package

The HTTP Request ID package provides utilities for generating, storing, and retrieving unique request identifiers for HTTP requests, enabling distributed tracing and request correlation.

## Features

- **Automatic ID Generation**: Generate unique identifiers for each request
- **Header Integration**: Extract and preserve IDs from request headers
- **Context Storage**: Store and retrieve request IDs via request context
- **Customizable Generation**: Configurable ID generation functions
- **Chain Compatibility**: Works seamlessly with middleware chains
- **Cross-Package Integration**: Works with `chain`, `handler`, and logging middleware

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
    idGenerator := func() string { return uuid.New().String() }
    handler := chain.Handler(
        mux,
        requestid.Handler("X-Request-Id", idGenerator),
    )
    http.ListenAndServe(":8080", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    requestID, exists := requestid.Context.Value(r.Context())
    if exists {
        w.Header().Set("X-Request-Id", requestID)
    }
    // ...
}
```

## API Reference

### Middleware

#### `Handler(header string, generator func() string) func(http.Handler) http.Handler`

Creates middleware for request ID generation and propagation.

### Context Access

#### `Context.Value(ctx context.Context) (string, bool)`

Retrieves request ID from context.

## Best Practices

- Use request IDs for distributed tracing and logging.
- Propagate request IDs across all middleware and handlers.
- Integrate with logging and error reporting systems.

## Related Packages

- [`chain`](../chain/README.md): Middleware chaining
- [`handler`](../handler/README.md): Base handler utilities
- [`auth`](../auth/README.md): Authentication middleware

## License

MIT
