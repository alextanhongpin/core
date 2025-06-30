# HTTP Response

The HTTP Response package provides a robust and standardized way to handle HTTP responses in Go applications. It offers structured JSON responses, error handling, response recording, and utilities for building consistent APIs.

## Features

- **Structured JSON Responses**: Standardized response format with data, error, and pagination information
- **Smart Error Handling**: Automatic error type detection and appropriate HTTP status code mapping
- **Response Recording**: Capture and inspect HTTP responses for middleware and testing
- **Pagination Support**: Built-in pagination metadata for list endpoints
- **Type Safety**: Generic support and clear interfaces

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/response"
)

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: "123", Name: "Alice"}
    
    // Send successful response
    response.OK(w, user)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: "456", Name: "Bob"}
    
    // Send successful response with custom status code
    response.OK(w, user, http.StatusCreated)
}
```

## Response Structure

All JSON responses follow a consistent structure:

```json
{
  "data": {},
  "error": {
    "code": "string",
    "message": "string", 
    "errors": {}
  },
  "pageInfo": {
    "hasPrevPage": false,
    "hasNextPage": true,
    "startCursor": "string",
    "endCursor": "string"
  }
}
```

## API Reference

### Success Responses

#### `OK(w http.ResponseWriter, data any, codes ...int)`

Sends a successful JSON response with the provided data.

```go
// Basic usage
response.OK(w, userData)

// With custom status code
response.OK(w, userData, http.StatusCreated)
```

#### `NoContent(w http.ResponseWriter)`

Sends a 204 No Content response.

```go
response.NoContent(w)
```

#### `JSON(w http.ResponseWriter, body any, codes ...int)`

Sends a JSON response with a custom body structure.

```go
response.JSON(w, &response.Body{
    Data: userData,
    PageInfo: &response.PageInfo{
        HasNextPage: true,
        EndCursor: "cursor123",
    },
})
```

### Error Responses

#### `ErrorJSON(w http.ResponseWriter, err error)`

Automatically handles error responses based on error type:

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    user, err := getUserByID(userID)
    if err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    response.OK(w, user)
}
```

**Error Type Handling:**

1. **Validation Errors**: Errors implementing `Map() map[string]any` interface
   - Status: 400 Bad Request
   - Includes field-level validation details

2. **Cause Errors**: `*cause.Error` types with custom codes
   - Status: Mapped from error code
   - Includes structured error information

3. **Unknown Errors**: All other error types
   - Status: 500 Internal Server Error
   - Generic error message for security

### Response Recording

The `ResponseWriterRecorder` allows you to capture response data for middleware, logging, or testing:

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        recorder := response.NewResponseWriterRecorder(w)
        recorder.SetWriteBody(true)
        
        next.ServeHTTP(recorder, r)
        
        // Access response data
        statusCode := recorder.StatusCode()
        body := recorder.Body()
        
        log.Printf("Response: %d, Body: %s", statusCode, body)
    })
}
```

#### Methods

- `NewResponseWriterRecorder(w http.ResponseWriter) *ResponseWriterRecorder`: Create a new recorder
- `SetWriteBody(bool)`: Enable/disable body recording
- `StatusCode() int`: Get the response status code
- `Body() []byte`: Get a copy of the response body
- `Unwrap() http.ResponseWriter`: Get the underlying ResponseWriter

### Response Reading

#### `Read(w *http.Response) ([]byte, error)`

Reads response body while preserving it for subsequent reads:

```go
client := &http.Client{}
resp, err := client.Get("https://api.example.com/users")
if err != nil {
    return err
}
defer resp.Body.Close()

// Read body without consuming it
body, err := response.Read(resp)
if err != nil {
    return err
}

// Body can still be read again
// resp.Body is restored with the same content
```

## Error Handling Examples

### Custom Application Errors

```go
import "github.com/alextanhongpin/errors/cause"
import "github.com/alextanhongpin/errors/codes"

func GetUser(id string) (*User, error) {
    if id == "" {
        return nil, cause.New(codes.BadRequest, "invalid_user_id", "User ID is required")
    }
    
    user, err := db.FindUser(id)
    if err == sql.ErrNoRows {
        return nil, cause.New(codes.NotFound, "user_not_found", "User not found")
    }
    
    return user, err
}

func Handler(w http.ResponseWriter, r *http.Request) {
    user, err := GetUser(r.URL.Query().Get("id"))
    if err != nil {
        response.ErrorJSON(w, err) // Automatically maps to appropriate status code
        return
    }
    
    response.OK(w, user)
}
```

### Validation Errors

```go
type CreateUserRequest struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

func (r CreateUserRequest) Validate() error {
    errs := cause.Map{}
    
    if r.Email == "" {
        errs["email"] = cause.Required("email", r.Email)
    } else if !strings.Contains(r.Email, "@") {
        errs["email"] = cause.Invalid("email", r.Email, "must be a valid email")
    }
    
    if r.Name == "" {
        errs["name"] = cause.Required("name", r.Name)
    }
    
    return errs.Err() // Returns nil if no errors
}
```

## Pagination

```go
func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
    users, pageInfo, err := getUsersWithPagination(r.URL.Query())
    if err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    response.JSON(w, &response.Body{
        Data: users,
        PageInfo: &response.PageInfo{
            HasPrevPage: pageInfo.HasPrev,
            HasNextPage: pageInfo.HasNext,
            StartCursor: pageInfo.StartCursor,
            EndCursor:   pageInfo.EndCursor,
        },
    })
}
```

## Best Practices

1. **Consistent Error Handling**: Always use `response.ErrorJSON()` for error responses to maintain consistency

2. **Status Code Selection**: Use appropriate HTTP status codes with `response.OK(w, data, statusCode)`

3. **Security**: Unknown errors automatically hide implementation details from clients

4. **Middleware Integration**: Use `ResponseWriterRecorder` for logging and monitoring middleware

5. **Validation**: Implement validation at the request level and let the response package handle error formatting

6. **Pagination**: Use the built-in pagination structure for list endpoints

## Integration with Request Package

This package works seamlessly with the request package for complete HTTP handling:

```go
import (
    "github.com/alextanhongpin/core/http/request"
    "github.com/alextanhongpin/core/http/response"
)

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := request.DecodeJSON(r, &req); err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    user, err := createUser(req)
    if err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    response.OK(w, user, http.StatusCreated)
}
```
