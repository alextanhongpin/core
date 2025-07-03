# HTTP ContextKey Package

The HTTP ContextKey package provides type-safe context key management for Go web applications, enabling secure and type-checked storage and retrieval of values in request contexts.

## Features

- **Type Safety**: Generic type parameters for compile-time type checking
- **Panic-Free Operations**: Safe value retrieval without type assertions
- **Existence Checking**: Easy detection of missing context values
- **Must Pattern**: Optional panic-based retrieval for critical values
- **Error Handling**: Clear error messages for debugging

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
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
        
        // Use the value
        fmt.Fprintf(w, "Hello, %s", userID)
    })
    
    // Middleware to set context values
    http.ListenAndServe(":8080", authMiddleware(http.DefaultServeMux))
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set values in context
        ctx := r.Context()
        ctx = UserIDKey.WithValue(ctx, "user-123")
        ctx = UserRolesKey.WithValue(ctx, []string{"admin", "user"})
        ctx = RequestTimeKey.WithValue(ctx, time.Now())
        
        // Call next handler with updated context
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## API Reference

### Context Key Definition

#### `Key[T any]`

A strongly-typed context key for values of type `T`.

```go
// Define typed context keys
var UserKey contextkey.Key[*User] = "user"
var TokenKey contextkey.Key[*jwt.Token] = "token"
var IsAuthenticatedKey contextkey.Key[bool] = "is_authenticated"
```

### Setting Context Values

#### `(k Key[T]) WithValue(ctx context.Context, t T) context.Context`

Sets a value in the context with type safety.

```go
// Set a value in the context
ctx = UserKey.WithValue(ctx, user)
```

### Getting Context Values

#### `(k Key[T]) Value(ctx context.Context) (T, bool)`

Safely retrieves a value from the context with type checking.

```go
// Get a value from context
user, exists := UserKey.Value(ctx)
if !exists {
    // Handle missing value
}
```

#### `(k Key[T]) MustValue(ctx context.Context) T`

Retrieves a value from the context, panicking if it doesn't exist.

```go
// Get a value that must exist
user := UserKey.MustValue(ctx) // Panics if not found
```

## Error Handling

The package defines an error for missing context keys:

```go
var ErrNotFound = errors.New("contextkey: key not found")
```

When using `MustValue`, panics include this error with the key name:

```go
// If UserKey doesn't exist in the context
user := UserKey.MustValue(ctx)
// Panics with: "contextkey: key not found: user"
```

## Common Usage Patterns

### Authentication Data

Store authentication data securely in the request context:

```go
// Define keys
var (
    UserIDKey contextkey.Key[string] = "user_id"
    UserRolesKey contextkey.Key[[]string] = "user_roles"
    IsAdminKey contextkey.Key[bool] = "is_admin"
)

// Authentication middleware
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token, err := extractAndValidateToken(r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Set auth data in context
        ctx := r.Context()
        ctx = UserIDKey.WithValue(ctx, token.Subject)
        ctx = UserRolesKey.WithValue(ctx, token.Roles)
        ctx = IsAdminKey.WithValue(ctx, hasAdminRole(token.Roles))
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Access in handlers
func adminOnlyHandler(w http.ResponseWriter, r *http.Request) {
    isAdmin, _ := IsAdminKey.Value(r.Context())
    if !isAdmin {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Admin-only operations...
}
```

### Request Metadata

Store request metadata for logging or tracing:

```go
// Define keys
var (
    RequestIDKey contextkey.Key[string] = "request_id"
    StartTimeKey contextkey.Key[time.Time] = "start_time"
)

// Middleware to set request metadata
func requestMetadataMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        startTime := time.Now()
        
        ctx := r.Context()
        ctx = RequestIDKey.WithValue(ctx, requestID)
        ctx = StartTimeKey.WithValue(ctx, startTime)
        
        // Add request ID to response headers
        w.Header().Set("X-Request-ID", requestID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Optional vs Required Values

Choose the right retrieval method based on necessity:

```go
// For optional values
traceID, exists := TraceIDKey.Value(ctx)
if exists {
    // Use trace ID for logging
}

// For required values (will panic if missing)
userID := UserIDKey.MustValue(ctx)
```

## Best Practices

1. **Key Naming**: Use descriptive key names for clarity and debugging
2. **Type Selection**: Choose appropriate types for context values
3. **Limited Data**: Store only essential data in the context
4. **Package Grouping**: Define related keys in the same package
5. **Scope Control**: Use unexported variables for internal context keys

## Comparison with Standard Context

The contextkey package improves on Go's standard context in several ways:

```go
// Standard context approach (unsafe)
userID, ok := ctx.Value("user_id").(string)
if !ok {
    // Type assertion could fail
}

// Contextkey approach (type-safe)
userID, exists := UserIDKey.Value(ctx)
if !exists {
    // No type assertion, compile-time safety
}
```
