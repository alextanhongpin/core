# HTTP Core Library

A comprehensive Go HTTP utilities library providing essential building blocks for building robust web applications and APIs. This library offers a collection of well-tested, production-ready modules that handle common HTTP concerns with best practices baked in.

## Overview

This library provides a modular approach to HTTP handling in Go, with each package focused on a specific concern. All modules are designed to work together seamlessly while remaining independent and composable.

## Features

- **🔐 Authentication & Authorization**: JWT, Basic Auth, and Bearer token handling
- **📝 Request/Response Handling**: JSON parsing, validation, and standardized responses
- **🔗 Middleware Chaining**: Composable middleware patterns
- **🛡️ Security**: Webhook verification, request ID tracking
- **📄 Template Engine**: Hot-reloadable template system
- **🚀 Server Management**: Graceful shutdown and zero-downtime deployments
- **📊 Pagination**: Cursor-based pagination utilities
- **⚡ Production Ready**: Comprehensive test coverage and battle-tested components

## Modules

### 🔐 Auth (`./auth`)

Comprehensive authentication and authorization utilities supporting multiple authentication methods.

**Features:**
- JWT token generation and verification
- Basic HTTP authentication
- Bearer token authentication
- Context-based claims storage
- Middleware for protecting routes

**Quick Start:**
```go
// JWT Authentication
jwt := auth.NewJWT([]byte("secret"))
token, _ := jwt.Sign(auth.Claims{Subject: "user@example.com"}, time.Hour)

// Bearer Token Middleware
handler = auth.BearerHandler(handler, []byte("secret"))

// Basic Auth Middleware  
credentials := map[string]string{"admin": "password"}
handler = auth.BasicHandler(handler, credentials)
```

### 🔗 Chain (`./chain`)

Elegant middleware composition for building request processing pipelines.

**Features:**
- Clean middleware chaining syntax
- Support for both `http.Handler` and `http.HandlerFunc`
- Proper execution order (first middleware wraps all others)

**Quick Start:**
```go
handler := chain.Handler(
    myHandler,
    loggingMiddleware,
    authMiddleware,
    rateLimitMiddleware,
)
```

### 🔑 Context Key (`./contextkey`)

Type-safe context key management for passing data through request chains.

**Features:**
- Generic type safety
- Panic-safe value retrieval
- Context value existence checking

**Quick Start:**
```go
var UserContext contextkey.Key[*User] = "user"

// Store value
ctx = UserContext.WithValue(ctx, user)

// Retrieve value
user, exists := UserContext.Value(ctx)
```

### 🎯 Handler (`./handler`)

Base handler with common functionality for building HTTP endpoints.

**Features:**  
- JSON request/response handling
- Integrated validation
- Structured error handling
- Request logging

**Quick Start:**
```go
type Controller struct {
    handler.BaseHandler
}

func (c *Controller) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := c.ReadJSON(r, &req); err != nil {
        c.Next(w, r, err)
        return
    }
    
    user, err := c.userService.Create(req)
    if err != nil {
        c.Next(w, r, err)
        return
    }
    
    c.OK(w, user, http.StatusCreated)
}
```

### 📊 Pagination (`./pagination`)

Cursor-based pagination for efficient large dataset handling.

**Features:**
- Cursor-based pagination
- Automatic has-next detection
- Database-friendly limit handling

**Quick Start:**
```go
cursor := &pagination.Cursor[int]{First: 10}
paginated := pagination.Paginate(items, cursor)

// Use cursor.Limit() for database queries
limit := cursor.Limit() // Returns 11 (First + 1)
```

### 📨 Request (`./request`)

Request parsing and validation utilities.

**Features:**
- JSON body parsing with validation
- URL parameter extraction (query, path, form)
- Type-safe value conversion
- Base64 encoding/decoding utilities
- Request body reading and cloning

**Quick Start:**
```go
// JSON parsing with validation
var user CreateUserRequest
if err := request.DecodeJSON(r, &user); err != nil {
    // Handle validation errors
}

// Parameter extraction
userID := request.PathValue(r, "id").Int64()
page := request.QueryValue(r, "page").IntN(1) // Default to 1
cursor := request.QueryValue(r, "cursor").FromBase64().String()
```

### 🔍 Request ID (`./requestid`)

Request ID generation and tracking for distributed tracing.

**Features:**
- Automatic request ID generation
- Header-based ID preservation
- Context integration

**Quick Start:**
```go
handler = requestid.Handler(handler, "X-Request-Id", func() string {
    return uuid.New().String()
})

// Access in handlers
reqID, _ := requestid.Context.Value(ctx)
```

### 📤 Response (`./response`)

Standardized HTTP response handling with comprehensive error management.

**Features:**
- Structured JSON responses
- Automatic error type detection
- HTTP status code mapping
- Response recording for middleware
- Pagination metadata support

**Quick Start:**
```go
// Success responses
response.OK(w, userData)
response.OK(w, userData, http.StatusCreated)

// Error handling
response.ErrorJSON(w, err) // Automatic status code mapping

// Custom responses
response.JSON(w, &response.Body{
    Data: users,
    PageInfo: &response.PageInfo{HasNextPage: true},
})
```

### 🖥️ Server (`./server`)

Production-ready HTTP server with graceful shutdown and advanced features.

**Features:**
- Graceful shutdown handling
- Zero-downtime deployments (with `forever.go`)
- Signal handling (SIGTERM, SIGINT)
- Reasonable default timeouts
- Multi-server support

**Quick Start:**
```go
// Simple server
server.ListenAndServe(":8080", handler)

// Advanced server with graceful shutdown
srv := server.New(":8080", handler)
server.WaitGroup(srv)

// Zero-downtime deployments
server.ListenAndServeForever(":8080", handler)
```

### 📄 Template (`./templ`)

Flexible template engine with hot-reload support and composition capabilities.

**Features:**
- Hot-reload for development
- Template composition and extension
- Embed.FS support
- Function map support
- Compile-time template validation

**Quick Start:**
```go
tpl := &templ.Template{
    FS: os.DirFS("templates"),
    HotReload: true,
}

// Compile templates
homePage := tpl.Compile("base.html", "home.html")
homePage.Execute(w, data)

// Template extension
base := tpl.Compile("base.html", "partials/*.html")
home := base.Extend("home.html")
```

### 🔐 Webhook (`./webhook`)

Secure webhook handling with signature verification.

**Features:**
- HMAC-SHA256 signature verification
- Multi-secret support for key rotation
- Automatic request signing
- Timestamp-based verification

**Quick Start:**
```go
// Webhook handler
handler = webhook.Handler(myHandler, []byte("secret"))

// Multi-secret support (for key rotation)
handler = webhook.Handler(myHandler, oldSecret, newSecret)

// Manual signature verification
payload := webhook.NewPayload(body)
signature := payload.Sign(secret)
isValid := payload.Verify(signature, secret)
```

## Installation

```bash
go get github.com/alextanhongpin/core/http
```

## Usage Patterns

### Complete API Endpoint Example

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/http/auth"
    "github.com/alextanhongpin/core/http/chain"
    "github.com/alextanhongpin/core/http/handler"
    "github.com/alextanhongpin/core/http/request"
    "github.com/alextanhongpin/core/http/requestid"
    "github.com/alextanhongpin/core/http/server"
)

type UserController struct {
    handler.BaseHandler
}

type CreateUserRequest struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

func (r CreateUserRequest) Validate() error {
    // Validation logic here
    return nil
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := c.ReadJSON(r, &req); err != nil {
        c.Next(w, r, err)
        return
    }
    
    // Business logic
    user := &User{Email: req.Email, Name: req.Name}
    
    c.OK(w, user, http.StatusCreated)
}

func main() {
    controller := &UserController{}
    
    mux := http.NewServeMux()
    mux.HandleFunc("POST /users", controller.CreateUser)
    
    // Apply middleware chain
    handler := chain.Handler(
        mux,
        requestid.Handler("X-Request-Id", generateID),
        auth.BearerHandler([]byte("secret")),
    )
    
    server.ListenAndServe(":8080", handler)
}
```

### Error Handling Pattern

```go
// Custom application errors
func GetUser(id string) (*User, error) {
    if id == "" {
        return nil, cause.New(codes.BadRequest, "invalid_id", "User ID is required")
    }
    
    user, err := db.FindUser(id)
    if err == sql.ErrNoRows {
        return nil, cause.New(codes.NotFound, "user_not_found", "User not found")
    }
    
    return user, err
}

// Handler automatically maps errors to appropriate HTTP status codes
func (c *Controller) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := request.PathValue(r, "id").String()
    
    user, err := c.userService.GetUser(userID)
    if err != nil {
        c.Next(w, r, err) // Automatic error handling
        return
    }
    
    c.OK(w, user)
}
```

## Testing

The library includes comprehensive test coverage with HTTP test dumps for verification:

```go
func TestCreateUser(t *testing.T) {
    w := httptest.NewRecorder()
    r := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"john"}`))
    
    httpdump.Handler(t, handler).ServeHTTP(w, r)
    
    // Test dumps are automatically generated and compared
}
```

## Configuration

### Environment-based Configuration

```go
config := &server.Config{
    Port: getEnv("PORT", ":8080"),
    ReadTimeout: getDuration("READ_TIMEOUT", 5*time.Second),
    WriteTimeout: getDuration("WRITE_TIMEOUT", 5*time.Second),
}
```

### Production Deployment

```go
// Zero-downtime deployment
server.ListenAndServeForever(":8080", handler)

// Send SIGUSR2 to trigger upgrade
// kill -SIGUSR2 $(lsof -ti:8080)
```

## Best Practices

1. **Error Handling**: Use structured errors with the `response.ErrorJSON()` function
2. **Middleware Order**: Apply middleware in logical order (logging → auth → rate limiting → handler)
3. **Request Validation**: Implement the `Validate()` method on request structures
4. **Context Usage**: Use type-safe context keys for passing data between middleware
5. **Testing**: Use HTTP test dumps for comprehensive endpoint testing
6. **Security**: Always validate and sanitize input, use HTTPS in production

## Dependencies

- `github.com/golang-jwt/jwt/v5` - JWT token handling
- `github.com/google/uuid` - UUID generation
- `github.com/alextanhongpin/errors` - Structured error handling
- Standard library packages for HTTP handling

## Contributing

This library follows Go best practices and maintains high test coverage. When contributing:

1. Add comprehensive tests for new functionality
2. Update documentation and examples
3. Follow the existing code style and patterns
4. Ensure backward compatibility

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Performance

The library is designed for production use with:
- Zero allocation paths where possible
- Efficient request/response handling
- Minimal memory overhead
- Proper resource cleanup

## Support

For questions, issues, or contributions, please refer to the GitHub repository and documentation.
