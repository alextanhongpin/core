# Handlers - Testable Request/Response Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/handlers.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/handlers)

Package `handlers` provides a lightweight, testable HTTP-like request/response pattern for internal service communication, message processing, and testing. It abstracts the HTTP layer to enable easier unit testing and decoupled service architectures.

## Features

- **HTTP-like Interface**: Familiar request/response pattern without HTTP dependencies
- **Middleware Support**: Composable middleware for cross-cutting concerns
- **Context Support**: Full context.Context integration for cancellation and timeouts
- **Testable Design**: Easy to unit test without HTTP servers
- **Metadata Support**: Request and response metadata for headers, tracing, etc.
- **Type Safety**: Strongly typed request/response handling
- **Zero HTTP Dependencies**: Pure Go implementation for internal use

## Installation

```bash
go get github.com/alextanhongpin/core/types/handlers
```

## Quick Start

```go
package main

import (
    "fmt"
    "strings"
    "github.com/alextanhongpin/core/types/handlers"
)

func main() {
    router := handlers.NewRouter()

    // Register a handler
    router.HandleFunc("greet", func(w handlers.ResponseWriter, r *handlers.Request) error {
        type Request struct {
            Name string `json:"name"`
        }
        
        var req Request
        if err := r.Decode(&req); err != nil {
            w.WriteStatus(400)
            return w.Encode(map[string]string{"error": "invalid request"})
        }

        return w.Encode(map[string]string{
            "message": fmt.Sprintf("Hello, %s!", req.Name),
        })
    })

    // Process a request
    req := handlers.NewRequest("greet", strings.NewReader(`{"name": "Alice"}`))
    resp, err := router.Do(req)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Status: %d\n", resp.Status)
    fmt.Printf("Response: %s\n", resp.String())
    // Output:
    // Status: 200
    // Response: {"message":"Hello, Alice!"}
}
```

## Core Concepts

### Request

The `Request` type represents an incoming request with pattern matching, body, metadata, and context:

```go
type Request struct {
    Pattern   string            // Route pattern
    Body      io.Reader         // Request body
    Meta      map[string]string // Request metadata
    Timestamp time.Time         // Request timestamp
}

// Create and configure requests
req := handlers.NewRequest("user.create", body)
req.WithMeta("user_id", "123")
req.WithContext(ctx)
```

### Response

The `Response` type provides structured response handling:

```go
type Response struct {
    Body      *bytes.Buffer     // Response body
    Status    int               // HTTP-like status code
    Headers   map[string]string // Response headers
    Meta      map[string]string // Response metadata
    Timestamp time.Time         // Response timestamp
    Duration  time.Duration     // Processing duration
}

// Use in handlers
func myHandler(w handlers.ResponseWriter, r *handlers.Request) error {
    w.WriteStatus(201)
    w.SetHeader("Content-Type", "application/json")
    w.SetMeta("trace_id", "abc123")
    return w.Encode(responseData)
}
```

### Router

The `Router` manages routing and middleware:

```go
router := handlers.NewRouter()
router.WithTimeout(30 * time.Second)
router.Use(middleware1, middleware2)
router.HandleFunc("pattern", handlerFunc)
```

## Middleware

### Built-in Middleware

#### Logging Middleware
```go
router.Use(handlers.LoggingMiddleware(func(pattern string, duration time.Duration, status int) {
    log.Printf("[%s] %d - %v", pattern, status, duration)
}))
```

#### Recovery Middleware
```go
router.Use(handlers.RecoveryMiddleware())
```

#### Authentication Middleware
```go
router.Use(handlers.AuthMiddleware(func(token string) error {
    return validateToken(token)
}))
```

#### Rate Limiting Middleware
```go
router.Use(handlers.RateLimitMiddleware(func(pattern string) bool {
    return rateLimiter.Allow(pattern)
}))
```

### Custom Middleware

```go
func CustomMiddleware() handlers.Middleware {
    return func(next handlers.Handler) handlers.Handler {
        return handlers.HandlerFunc(func(w handlers.ResponseWriter, r *handlers.Request) error {
            // Pre-processing
            start := time.Now()
            
            // Call next handler
            err := next.Handle(w, r)
            
            // Post-processing
            duration := time.Since(start)
            w.SetMeta("processing_time", duration.String())
            
            return err
        })
    }
}
```

## Real-World Examples

### User Service

```go
type UserService struct {
    users map[string]User
    router *handlers.Router
}

func NewUserService() *UserService {
    us := &UserService{
        users: make(map[string]User),
        router: handlers.NewRouter(),
    }
    
    // Add middleware
    us.router.Use(handlers.LoggingMiddleware(logFunc))
    us.router.Use(handlers.AuthMiddleware(us.validateToken))
    
    // Register handlers
    us.router.HandleFunc("users.create", us.createUser)
    us.router.HandleFunc("users.get", us.getUser)
    us.router.HandleFunc("users.list", us.listUsers)
    
    return us
}

func (us *UserService) createUser(w handlers.ResponseWriter, r *handlers.Request) error {
    var req CreateUserRequest
    if err := r.Decode(&req); err != nil {
        w.WriteStatus(400)
        return w.Encode(map[string]string{"error": "invalid request"})
    }

    user := User{
        ID:    generateID(),
        Name:  req.Name,
        Email: req.Email,
    }
    us.users[user.ID] = user

    w.WriteStatus(201)
    w.SetHeader("Location", fmt.Sprintf("/users/%s", user.ID))
    return w.Encode(user)
}

func (us *UserService) ProcessRequest(req *handlers.Request) (*handlers.Response, error) {
    return us.router.Do(req)
}
```

### Message Queue Processor

```go
type MessageProcessor struct {
    router *handlers.Router
}

func NewMessageProcessor() *MessageProcessor {
    mp := &MessageProcessor{
        router: handlers.NewRouter(),
    }
    
    // Add middleware
    mp.router.Use(handlers.RecoveryMiddleware())
    mp.router.Use(handlers.LoggingMiddleware(logFunc))
    
    // Register message handlers
    mp.router.HandleFunc("user.created", mp.handleUserCreated)
    mp.router.HandleFunc("order.placed", mp.handleOrderPlaced)
    mp.router.HandleFunc("email.send", mp.handleEmailSend)
    
    return mp
}

func (mp *MessageProcessor) ProcessMessage(msgType string, payload []byte) error {
    req := handlers.NewRequest(msgType, bytes.NewReader(payload))
    req.WithMeta("timestamp", time.Now().Format(time.RFC3339))
    
    _, err := mp.router.Do(req)
    return err
}

func (mp *MessageProcessor) handleUserCreated(w handlers.ResponseWriter, r *handlers.Request) error {
    var msg UserCreatedMessage
    if err := r.Decode(&msg); err != nil {
        return err
    }
    
    // Process user creation
    log.Printf("Setting up profile for user %s", msg.UserID)
    
    return w.Encode(map[string]string{"status": "processed"})
}
```

### API Testing Framework

```go
type APITest struct {
    router *handlers.Router
}

func NewAPITest() *APITest {
    return &APITest{
        router: handlers.NewRouter(),
    }
}

func (at *APITest) RegisterEndpoint(pattern string, handler handlers.HandlerFunc) {
    at.router.HandleFunc(pattern, handler)
}

func (at *APITest) GET(pattern string, meta map[string]string) (*handlers.Response, error) {
    req := handlers.NewRequest(pattern, strings.NewReader("{}"))
    for k, v := range meta {
        req.WithMeta(k, v)
    }
    return at.router.Do(req)
}

func (at *APITest) POST(pattern string, body string, meta map[string]string) (*handlers.Response, error) {
    req := handlers.NewRequest(pattern, strings.NewReader(body))
    for k, v := range meta {
        req.WithMeta(k, v)
    }
    return at.router.Do(req)
}

// Usage in tests
func TestUserAPI(t *testing.T) {
    test := NewAPITest()
    test.RegisterEndpoint("users.create", createUserHandler)
    
    resp, err := test.POST("users.create", `{"name":"John","email":"john@example.com"}`, nil)
    assert.NoError(t, err)
    assert.Equal(t, 201, resp.Status)
}
```

### Internal RPC System

```go
type RPCServer struct {
    router *handlers.Router
}

func NewRPCServer() *RPCServer {
    rpc := &RPCServer{
        router: handlers.NewRouter().WithTimeout(10 * time.Second),
    }
    
    // Add tracing middleware
    rpc.router.Use(func(next handlers.Handler) handlers.Handler {
        return handlers.HandlerFunc(func(w handlers.ResponseWriter, r *handlers.Request) error {
            traceID := generateTraceID()
            r.WithMeta("trace_id", traceID)
            w.SetMeta("trace_id", traceID)
            return next.Handle(w, r)
        })
    })
    
    return rpc
}

func (rpc *RPCServer) RegisterService(serviceName string, methods map[string]handlers.HandlerFunc) {
    for method, handler := range methods {
        pattern := fmt.Sprintf("%s.%s", serviceName, method)
        rpc.router.HandleFunc(pattern, handler)
    }
}

func (rpc *RPCServer) Call(service, method string, request interface{}) (*handlers.Response, error) {
    body, _ := json.Marshal(request)
    pattern := fmt.Sprintf("%s.%s", service, method)
    
    req := handlers.NewRequest(pattern, bytes.NewReader(body))
    return rpc.router.Do(req)
}
```

## Context and Timeouts

### Request Timeout

```go
// Set router-level timeout
router := handlers.NewRouter().WithTimeout(30 * time.Second)

// Per-request timeout
ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
defer cancel()

req := handlers.NewRequest("slow-operation", body).WithContext(ctx)
resp, err := router.Do(req)
if err == handlers.ErrRequestTimeout {
    // Handle timeout
}
```

### Context Values

```go
// Add context values
ctx := context.WithValue(context.Background(), "user_id", "123")
req := handlers.NewRequest("protected", body).WithContext(ctx)

// Access in handler
func protectedHandler(w handlers.ResponseWriter, r *handlers.Request) error {
    userID := r.Context().Value("user_id").(string)
    // Use userID...
}
```

## Testing

### Unit Testing Handlers

```go
func TestCreateUser(t *testing.T) {
    service := NewUserService()
    
    req := handlers.NewRequest("users.create", 
        strings.NewReader(`{"name":"John","email":"john@example.com"}`))
    req.WithMeta("Authorization", "Bearer valid-token")
    
    resp, err := service.ProcessRequest(req)
    
    assert.NoError(t, err)
    assert.Equal(t, 201, resp.Status)
    
    var user User
    err = resp.Decode(&user)
    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
}
```

### Integration Testing

```go
func TestUserWorkflow(t *testing.T) {
    service := NewUserService()
    
    // Create user
    createReq := handlers.NewRequest("users.create", 
        strings.NewReader(`{"name":"John","email":"john@example.com"}`))
    createResp, err := service.ProcessRequest(createReq)
    require.NoError(t, err)
    require.Equal(t, 201, createResp.Status)
    
    // Extract user ID
    var user User
    err = createResp.Decode(&user)
    require.NoError(t, err)
    
    // Get user
    getReq := handlers.NewRequest("users.get", strings.NewReader("{}"))
    getReq.WithMeta("user_id", user.ID)
    getResp, err := service.ProcessRequest(getReq)
    require.NoError(t, err)
    require.Equal(t, 200, getResp.Status)
}
```

## Best Practices

1. **Use Consistent Patterns**: Establish naming conventions for your patterns (e.g., `service.method`)

2. **Handle Errors Gracefully**: Always set appropriate status codes and return structured error responses

3. **Leverage Middleware**: Use middleware for cross-cutting concerns like logging, auth, and recovery

4. **Test Without HTTP**: Write unit tests that don't require HTTP servers

5. **Use Context**: Leverage context for timeouts, cancellation, and request-scoped data

6. **Structure Responses**: Return consistent response formats across your application

7. **Validate Input**: Always validate and sanitize request data

8. **Set Timeouts**: Configure appropriate timeouts for your use case

## Use Cases

- **Microservice Communication**: Internal service-to-service communication
- **Message Processing**: Queue message handlers and processors  
- **API Testing**: Testing API logic without HTTP overhead
- **Event Handling**: Domain event processing systems
- **RPC Systems**: Internal RPC-like communication
- **Plugin Systems**: Pluggable handler architectures
- **Command Processing**: Command pattern implementations

## Migration from HTTP

Converting from HTTP handlers is straightforward:

```go
// HTTP handler
func httpHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP-specific code
}

// Handlers equivalent
func internalHandler(w handlers.ResponseWriter, r *handlers.Request) error {
    // Same logic, different interfaces
    return nil
}
```

## Performance Considerations

- **No HTTP Overhead**: Faster than HTTP for internal communication
- **Memory Efficient**: Reuses buffers and minimal allocations
- **Concurrent Safe**: Safe for concurrent use across goroutines
- **Timeout Support**: Built-in timeout handling prevents hanging operations

## License

MIT License - see LICENSE file for details.
