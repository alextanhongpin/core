# HTTP ContextKey Package

The HTTP ContextKey package provides type-safe context key management for Go web applications, enabling secure and type-checked storage and retrieval of values in request contexts.

## Features

- **Type Safety**: Generic type parameters for compile-time type checking
- **Panic-Free Operations**: Safe value retrieval without type assertions
- **Existence Checking**: Easy detection of missing context values
- **Must Pattern**: Optional panic-based retrieval for critical values
- **Error Handling**: Clear error messages for debugging
- **Cross-Package Integration**: Works seamlessly with `auth`, `requestid`, and other middleware

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "github.com/alextanhongpin/core/http/contextkey"
)

// Define strongly typed context keys
var (
    UserIDKey contextkey.Key[string] = "user_id"
    UserRolesKey contextkey.Key[[]string] = "user_roles"
    RequestTimeKey contextkey.Key[time.Time] = "request_time"
)

func main() {
    http.HandleFunc("/api/resource", func(w http.ResponseWriter, r *http.Request) {
        // Get values from context
        userID, exists := UserIDKey.Value(r.Context())
        if !exists {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        fmt.Fprintf(w, "Hello, %s", userID)
    })
    // Middleware to set context values
    http.ListenAndServe(":8080", authMiddleware(http.DefaultServeMux))
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = UserIDKey.WithValue(ctx, "user-123")
        ctx = UserRolesKey.WithValue(ctx, []string{"admin", "user"})
        ctx = RequestTimeKey.WithValue(ctx, time.Now())
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## API Reference

### Key Definition

```go
var UserIDKey contextkey.Key[string] = "user_id"
```

### Value Retrieval

```go
userID, exists := UserIDKey.Value(ctx)
```

### Must Pattern

```go
userID := UserIDKey.MustValue(ctx) // panics if missing
```

## Best Practices

- Use strongly typed keys for all context values.
- Prefer panic-free retrieval for non-critical values.
- Integrate with authentication and request ID middleware for full context propagation.

## Related Packages

- [`auth`](../auth/README.md): Authentication middleware
- [`requestid`](../requestid/README.md): Request ID propagation
- [`chain`](../chain/README.md): Middleware chaining

## License

MIT
