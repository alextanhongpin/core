# HTTP Authentication Package

The HTTP Authentication package provides comprehensive authentication utilities for Go web applications, including Basic Auth, Bearer token, and JWT authentication mechanisms. It is designed for production use, with flexible configuration, context integration, and custom error handling.

## Features
- Multiple authentication methods: Basic Auth (with SHA-256 hashed passwords), Bearer tokens, JWT
- Context-based claims and user storage
- Customizable error handling and optional authentication
- Middleware for route protection and claims propagation
- Type-safe context integration

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

    // JWT Authentication
    jwt := auth.NewJWT([]byte("your-secret-key"))
    token, _ := jwt.Sign(auth.Claims{Subject: "user@example.com"}, time.Hour)
    handler := auth.BearerHandler(mux, []byte("your-secret-key"))

    // Basic Authentication (store SHA-256 hashes)
    credentials := map[string]string{
        "admin": auth.HashPasswordSHA256("password123"),
        "user":  auth.HashPasswordSHA256("userpass"),
    }
    handler = auth.BasicHandler(handler, credentials)

    http.ListenAndServe(":8080", handler)
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := auth.ClaimsContext.Value(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    fmt.Fprintf(w, "Hello, %s", claims.Subject)
}
```

## API Reference

### JWT Authentication
- `NewJWT(secret []byte) *JWT` — Create a JWT handler
- `Sign(claims Claims, expiry time.Duration) (string, error)` — Sign claims
- `Verify(token string) (Claims, error)` — Verify JWT token

### Bearer Token Middleware
- `BearerHandler(next http.Handler, secret []byte) http.Handler` — Middleware for JWT Bearer tokens
- `RequireBearerHandler(next http.Handler) http.Handler` — Enforce authentication on protected routes

### Basic Authentication
- `BasicHandler(next http.Handler, credentials map[string]string) http.Handler` — Middleware for Basic Auth (credentials must be SHA-256 hashes)
- `HashPasswordSHA256(password string) string` — Utility to hash passwords for storage

### Context Utilities
- `ClaimsContext` — Type-safe context key for JWT claims
- `UsernameContext` — Type-safe context key for Basic Auth username

## Best Practices
- Use context keys for claims and user data
- Always validate and sanitize input
- Store Basic Auth passwords as SHA-256 hashes using `HashPasswordSHA256`
- Customize error handling for better UX
- Prefer JWT for stateless APIs, Basic Auth for simple admin panels

## See Also
- [`chain`](../chain/README.md) — Middleware composition
- [`handler`](../handler/README.md) — Base handler patterns
- [`contextkey`](../contextkey/README.md) — Type-safe context keys
