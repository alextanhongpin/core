# HTTP Authentication Package

The HTTP Authentication package provides comprehensive authentication utilities for web applications, including Basic Auth, Bearer token, and JWT authentication mechanisms.

## Features

- **Multiple Authentication Methods**: Basic Auth, Bearer tokens, and JWT support
- **Context Integration**: Store and retrieve authentication data via request context
- **Custom Error Handling**: Configure custom error responses for auth failures
- **Optional Authentication**: Support for routes that allow but don't require auth
- **Flexible Configuration**: Customize behavior through configuration structures

## Quick Start

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/http/auth"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/resource", protectedHandler)
    
    // Choose your authentication method:
    
    // 1. JWT Authentication with Bearer Token
    jwtAuth := auth.NewJWT([]byte("your-secret-key"))
    handler := auth.RequireBearerHandler(mux, auth.BearerConfig{
        Secret: []byte("your-secret-key"),
    })
    
    // 2. Basic Authentication
    credentials := map[string]string{
        "admin": "password123",
        "user":  "userpass",
    }
    handler := auth.BasicHandler(mux, auth.BasicConfig{
        Credentials: credentials,
    })
    
    http.ListenAndServe(":8080", handler)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    // Access authenticated username
    username, exists := auth.UsernameFromContext(r.Context())
    if !exists {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    fmt.Fprintf(w, "Hello, %s", username)
}
```

## API Reference

### Basic Authentication

#### `BasicHandler(next http.Handler, config BasicConfig) http.Handler`

Creates middleware for HTTP Basic Authentication.

```go
// Configure basic auth
config := auth.BasicConfig{
    Credentials: map[string]string{
        "admin": "secret123",
        "user":  "password456",
    },
    Realm: "My Application", // Optional
    ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Access denied", http.StatusForbidden)
    }, // Optional
}

// Apply middleware
handler = auth.BasicHandler(handler, config)
```

### Bearer Token Authentication

#### `BearerHandler(next http.Handler, config BearerConfig) http.Handler`

Creates middleware for Bearer token authentication.

```go
// Configure bearer auth
config := auth.BearerConfig{
    Secret:      []byte("your-secret-key"),
    Required:    true, // Set to false for optional authentication
    ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
    }, // Optional
}

// Apply middleware
handler = auth.BearerHandler(handler, config)
```

#### `RequireBearerHandler(next http.Handler, config BearerConfig) http.Handler`

Shorthand for creating Bearer token middleware with required authentication.

### JWT Token Handling

#### `NewJWT(secret []byte, options ...Option) *JWT`

Creates a new JWT handler for token generation and verification.

```go
// Create a JWT handler with options
jwt := auth.NewJWT([]byte("secret"), 
    auth.WithSigningMethod(jwt.SigningMethodHS256),
    auth.WithIssuer("my-app"),
    auth.WithLeeway(30 * time.Second),
)

// Sign a token with claims
claims := auth.Claims{
    Subject: "user123",
    Email:   "user@example.com",
    Roles:   []string{"admin", "user"},
}
token, err := jwt.Sign(claims, 24*time.Hour)

// Verify a token
claims, err := jwt.Verify(token)
```

### Context Utilities

#### `UsernameFromContext(ctx context.Context) (string, bool)`

Retrieves the authenticated username from the context.

#### `ClaimsFromContext(ctx context.Context) (Claims, bool)`

Retrieves JWT claims from the context.

## Configuration

### Basic Authentication Configuration

```go
type BasicConfig struct {
    // Credentials maps usernames to passwords
    Credentials map[string]string
    
    // Realm is the authentication realm
    Realm string
    
    // ErrorHandler is called when authentication fails
    ErrorHandler func(w http.ResponseWriter, r *http.Request)
}
```

### Bearer Authentication Configuration

```go
type BearerConfig struct {
    // Secret is the signing secret for the token
    Secret []byte
    
    // Required determines if authentication is mandatory
    Required bool
    
    // ErrorHandler is called when authentication fails
    ErrorHandler func(w http.ResponseWriter, r *http.Request)
}
```

### JWT Configuration

```go
// Option functions for JWT configuration
jwt := auth.NewJWT(secret,
    auth.WithSigningMethod(jwt.SigningMethodHS384),
    auth.WithIssuer("api.myservice.com"),
    auth.WithLeeway(30 * time.Second),
)
```

## Best Practices

1. **Secret Management**: Store secrets securely and rotate them regularly
2. **Token Expiration**: Use appropriate expiration times for JWT tokens
3. **Error Responses**: Customize error handlers for proper security responses
4. **HTTPS**: Always use these authentication methods over HTTPS
5. **Token Validation**: Validate tokens with appropriate signature verification and claims checking

## Testing

The package includes test helpers and fixtures for authentication testing:

```go
// Create a test server with authentication
ts := httptest.NewServer(auth.BasicHandler(handler, auth.BasicConfig{
    Credentials: map[string]string{"test": "password"},
}))
defer ts.Close()

// Make authenticated request
req, _ := http.NewRequest("GET", ts.URL+"/api/resource", nil)
req.SetBasicAuth("test", "password")
resp, _ := http.DefaultClient.Do(req)
```

## Error Handling

The auth package provides sensible defaults for error handling but allows customization:

```go
// Custom error handler for authentication failures
errorHandler := func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "Authentication failed",
        "code":  "auth_required",
    })
}

config := auth.BearerConfig{
    Secret:       []byte("secret"),
    ErrorHandler: errorHandler,
}
```
