# HTTP Handler Package

The HTTP Handler package provides a base controller for HTTP handlers with common functionality for JSON handling, error management, response formatting, and structured logging.

## Features

- **JSON Request/Response Handling**: Simplified JSON parsing and response generation
- **Standardized Error Handling**: Consistent error responses with proper status codes
- **Structured Logging**: Integrated logging with request details
- **Response Formatting**: Consistent API response structures
- **Validation Integration**: Automatic validation of request structures
- **Controller Pattern**: Support for building REST controllers with shared functionality
- **Request Correlation**: Support for request ID tracking and correlation
- **Consistent Responses**: Standardized response formats across all endpoints
- **Cross-Package Integration**: Works with `request`, `response`, `requestid`, and `auth` packages

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

#### `ReadAndValidateJSON(r *http.Request, req interface{ Validate() error }) error`
Decodes and validates JSON request body.

### Error Handling

#### `Next(w http.ResponseWriter, r *http.Request, err error)`
Handles errors and sends standardized error responses.

### Response Formatting

#### `JSON(w http.ResponseWriter, v any, status ...int)`
Sends a JSON response with optional status code.

## Best Practices

- Use `BaseHandler` for all controllers to ensure consistent error and response handling.
- Integrate with `requestid` and `auth` middleware for full request context.
- Validate all incoming request payloads.

## Related Packages

- [`request`](../request/README.md): Request parsing and validation
- [`response`](../response/README.md): Response formatting
- [`requestid`](../requestid/README.md): Request ID propagation
- [`auth`](../auth/README.md): Authentication middleware
- [`chain`](../chain/README.md): Middleware chaining

## License

MIT
