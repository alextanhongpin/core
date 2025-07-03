# HTTP Handler Package

The HTTP Handler package provides a base controller for HTTP handlers with common functionality for JSON handling, error management, response formatting, and structured logging.

## Features

- **JSON Request/Response Handling**: Simplified JSON parsing and response generation
- **Standardized Error Handling**: Consistent error responses with proper status codes
- **Structured Logging**: Integrated logging with request details
- **Response Formatting**: Consistent API response structures
- **Validation Integration**: Automatic validation of request structures
- **Controller Pattern**: Support for building REST controllers with shared functionality
- **Structured Logging**: Built-in support for structured logging with request context
- **Validation**: Built-in validation support for request structs
- **Request Correlation**: Support for request ID tracking and correlation
- **Consistent Responses**: Standardized response formats across all endpoints

## Quick Start

### Basic Usage

```go
package main

import (
    "log/slog"
    "net/http"
    "github.com/alextanhongpin/core/http/handler"
)

type UserController struct {
    handler.BaseHandler
}

func NewUserController() *UserController {
    return &UserController{
        BaseHandler: handler.BaseHandler{}.WithLogger(slog.Default()),
    }
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := c.ReadAndValidateJSON(r, &req); err != nil {
        c.Next(w, r, err)
        return
    }
    
    // Business logic here...
    user := User{ID: 1, Name: req.Name}
    
    c.JSON(w, user, http.StatusCreated)
}
```

## Core Methods

### JSON Processing

#### `ReadJSON(r *http.Request, req any) error`
Decodes JSON from request body into the provided struct.

```go
var req CreateUserRequest
if err := c.ReadJSON(r, &req); err != nil {
    c.Next(w, r, err)
    return
}
```

#### `ReadAndValidateJSON(r *http.Request, req any) error`
Combines JSON reading and validation in one call.

```go
var req CreateUserRequest
if err := c.ReadAndValidateJSON(r, &req); err != nil {
    c.Next(w, r, err)
    return
}
```

### Validation

#### `Validate(req any) error`
Validates a request struct if it implements the `Validate() error` method.

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
    return cause.Map{
        "name":  cause.Required(r.Name),
        "email": cause.Required(r.Email),
    }.Err()
}
```

### Response Methods

#### `JSON(w http.ResponseWriter, data any, codes ...int)`
Writes a JSON response with optional status code.

```go
c.JSON(w, user)                        // 200 OK
c.JSON(w, user, http.StatusCreated)    // 201 Created
```

#### `OK(w http.ResponseWriter, data any, codes ...int)`
Alias for JSON method - writes successful JSON response.

```go
c.OK(w, users)                         // 200 OK
c.OK(w, user, http.StatusCreated)      // 201 Created
```

#### `NoContent(w http.ResponseWriter)`
Writes a 204 No Content response.

```go
c.NoContent(w) // 204 No Content
```

#### `ErrorJSON(w http.ResponseWriter, err error)`
Writes an error response in JSON format with appropriate status code.

```go
c.ErrorJSON(w, err)
```

### Error Handling

#### `Next(w http.ResponseWriter, r *http.Request, err error)`
Central error handling method that logs and responds to errors.

```go
if err := c.userService.GetUser(ctx, id); err != nil {
    c.Next(w, r, err)
    return
}
```

**Error Types Handled:**
- **Validation Errors**: Logged as warnings with 400 status
- **Cause Errors**: Logged as errors with mapped HTTP status codes
- **Generic Errors**: Logged as errors with 500 status

### Request Correlation

#### `GetRequestID(r *http.Request) string`
Extracts request ID from headers for correlation.

```go
requestID := c.GetRequestID(r) // Checks X-Request-ID, then X-Correlation-ID
```

#### `SetRequestID(w http.ResponseWriter, requestID string)`
Sets request ID in response headers.

```go
requestID := c.GetRequestID(r)
c.SetRequestID(w, requestID)
```

### Logger Configuration

#### `WithLogger(logger *slog.Logger) BaseHandler`
Configures the handler with a structured logger.

```go
base := handler.BaseHandler{}.WithLogger(slog.Default())
```

## Complete Example: User Management API

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "strconv"
    
    "github.com/alextanhongpin/core/http/handler"
    "github.com/alextanhongpin/errors/cause"
    "github.com/alextanhongpin/errors/codes"
)

// Domain models
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Request/Response DTOs
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
    return cause.Map{
        "name":  cause.Required(r.Name),
        "email": cause.Required(r.Email),
    }.Err()
}

type UpdateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r UpdateUserRequest) Validate() error {
    return cause.Map{
        "name":  cause.Required(r.Name),
        "email": cause.Required(r.Email),
    }.Err()
}

type UserResponse struct {
    User User `json:"user"`
}

type UsersResponse struct {
    Users []User `json:"users"`
    Total int    `json:"total"`
}

// Service layer
type UserService interface {
    Create(ctx context.Context, req CreateUserRequest) (User, error)
    GetByID(ctx context.Context, id int) (User, error)
    Update(ctx context.Context, id int, req UpdateUserRequest) (User, error)
    Delete(ctx context.Context, id int) error
    List(ctx context.Context) ([]User, error)
}

// Controller
type UserController struct {
    handler.BaseHandler
    userService UserService
}

func NewUserController(userService UserService) *UserController {
    return &UserController{
        BaseHandler: handler.BaseHandler{}.WithLogger(slog.Default()),
        userService: userService,
    }
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Extract and set request ID for correlation
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    
    // Read and validate request
    var req CreateUserRequest
    if err := c.ReadAndValidateJSON(r, &req); err != nil {
        c.Next(w, r, err)
        return
    }
    
    // Business logic
    user, err := c.userService.Create(r.Context(), req)
    if err != nil {
        c.Next(w, r, err)
        return
    }
    
    // Success response
    c.JSON(w, UserResponse{User: user}, http.StatusCreated)
}

func (c *UserController) GetUser(w http.ResponseWriter, r *http.Request) {
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    
    // Extract ID from URL (in real app, use router like gorilla/mux)
    idStr := r.URL.Query().Get("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.Next(w, r, cause.New(codes.InvalidArgument, "user/invalid_id", "Invalid user ID"))
        return
    }
    
    user, err := c.userService.GetByID(r.Context(), id)
    if err != nil {
        c.Next(w, r, err)
        return
    }
    
    c.OK(w, UserResponse{User: user})
}

func (c *UserController) UpdateUser(w http.ResponseWriter, r *http.Request) {
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    
    // Extract ID and validate request
    idStr := r.URL.Query().Get("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.Next(w, r, cause.New(codes.InvalidArgument, "user/invalid_id", "Invalid user ID"))
        return
    }
    
    var req UpdateUserRequest
    if err := c.ReadAndValidateJSON(r, &req); err != nil {
        c.Next(w, r, err)
        return
    }
    
    user, err := c.userService.Update(r.Context(), id, req)
    if err != nil {
        c.Next(w, r, err)
        return
    }
    
    c.JSON(w, UserResponse{User: user})
}

func (c *UserController) DeleteUser(w http.ResponseWriter, r *http.Request) {
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    
    idStr := r.URL.Query().Get("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.Next(w, r, cause.New(codes.InvalidArgument, "user/invalid_id", "Invalid user ID"))
        return
    }
    
    if err := c.userService.Delete(r.Context(), id); err != nil {
        c.Next(w, r, err)
        return
    }
    
    c.NoContent(w)
}

func (c *UserController) ListUsers(w http.ResponseWriter, r *http.Request) {
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    
    users, err := c.userService.List(r.Context())
    if err != nil {
        c.Next(w, r, err)
        return
    }
    
    c.OK(w, UsersResponse{
        Users: users,
        Total: len(users),
    })
}

// Router setup
func main() {
    userService := NewInMemoryUserService() // Your service implementation
    userController := NewUserController(userService)
    
    http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodPost:
            userController.CreateUser(w, r)
        case http.MethodGet:
            userController.ListUsers(w, r)
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })
    
    http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            userController.GetUser(w, r)
        case http.MethodPut:
            userController.UpdateUser(w, r)
        case http.MethodDelete:
            userController.DeleteUser(w, r)
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Best Practices

### 1. Always Use Request Correlation
```go
func (c *Controller) Handler(w http.ResponseWriter, r *http.Request) {
    requestID := c.GetRequestID(r)
    c.SetRequestID(w, requestID)
    // ... rest of handler
}
```

### 2. Validate Early and Often
```go
var req CreateUserRequest
if err := c.ReadAndValidateJSON(r, &req); err != nil {
    c.Next(w, r, err)
    return
}
```

### 3. Use Structured Error Handling
```go
if err != nil {
    c.Next(w, r, err) // Let BaseHandler handle logging and response
    return
}
```

### 4. Implement Validation Interface
```go
type Request struct {
    Field string `json:"field"`
}

func (r Request) Validate() error {
    return cause.Map{
        "field": cause.Required(r.Field),
    }.Err()
}
```

### 5. Use Appropriate Response Methods
```go
c.JSON(w, data, http.StatusCreated)  // For created resources
c.OK(w, data)                        // For successful gets/updates
c.NoContent(w)                       // For successful deletes
```

## Testing

The handler includes comprehensive test coverage. See `base_test.go` and `examples_test.go` for unit tests, integration tests, and benchmarks.

### Running Tests
```bash
go test -v ./...
go test -cover ./...
go test -bench=. ./...
```

## Error Handling Patterns

### Validation Errors (400 Bad Request)
```go
// Automatically handled by ReadAndValidateJSON
var req CreateUserRequest
if err := c.ReadAndValidateJSON(r, &req); err != nil {
    c.Next(w, r, err) // Results in 400 with validation details
    return
}
```

### Business Logic Errors
```go
// Use cause package for structured errors
if user.Email == existingEmail {
    err := cause.New(codes.Conflict, "user/email_exists", "Email already exists")
    c.Next(w, r, err) // Results in 409 Conflict
    return
}
```

### Generic Errors (500 Internal Server Error)
```go
if err := database.Save(user); err != nil {
    c.Next(w, r, err) // Results in 500 with error logged
    return
}
```

This documentation provides a comprehensive guide to using the BaseHandler effectively in real-world applications.
