# HTTP Request Package

The HTTP Request package provides utilities for parsing, validating, and accessing HTTP request data in Go web applications, with a focus on type-safe value handling.

## Features

- **JSON Body Parsing**: Decode and validate JSON request bodies
- **Value Extraction**: Get values from URL parameters, forms, and headers
- **Type-Safe Conversion**: Fluent API for converting string values to various types
- **Validation Support**: Integrated validation for request structures
- **Request Body Reading**: Read and restore request body content
- **Default Values**: Convenient handling of missing or empty values

## Quick Start

```go
package main

import (
    "net/http"
    
    "github.com/alextanhongpin/core/http/request"
    "github.com/alextanhongpin/core/http/response"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

// Implement validation for the request
func (r CreateUserRequest) Validate() error {
    // Your validation logic here
    return nil
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    // Parse JSON request with validation
    var req CreateUserRequest
    if err := request.DecodeJSON(r, &req); err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    // Process request...
    
    response.OK(w, map[string]string{"status": "success"})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    // Get path parameter and convert to int
    userID := request.PathValue(r, "id").Int()
    
    // Get query parameter with default value
    page := request.QueryValue(r, "page").IntN(1)
    
    // Get query parameter as time.Time
    since := request.QueryValue(r, "since").Time("2006-01-02")
    
    // Process request...
}
```

## API Reference

### JSON Handling

#### `DecodeJSON(r *http.Request, v any) error`

Decodes JSON request body into the provided struct and runs validation if available.

```go
var req UserRequest
if err := request.DecodeJSON(r, &req); err != nil {
    return err
}
```

### Value Extraction

#### `PathValue(r *http.Request, key string) Value`

Extracts a path parameter value from the request (compatible with path parameter extraction).

```go
userID := request.PathValue(r, "id").String()
```

#### `QueryValue(r *http.Request, key string) Value`

Extracts a query parameter value from the request URL.

```go
sort := request.QueryValue(r, "sort").String()
limit := request.QueryValue(r, "limit").IntN(10) // Default to 10
```

#### `FormValue(r *http.Request, key string) Value`

Extracts a form value from POST, PUT, or PATCH requests.

```go
email := request.FormValue(r, "email").String()
```

#### `HeaderValue(r *http.Request, key string) Value`

Extracts a header value from the request.

```go
userAgent := request.HeaderValue(r, "User-Agent").String()
```

### Type Conversion

The `Value` type provides fluent methods for safe type conversion:

```go
// String conversions
str := value.String()         // Get as string
optional := value.StringP()   // Get as *string (nil if empty)

// Integer conversions
i := value.Int()              // Get as int
i64 := value.Int64()          // Get as int64
iDef := value.IntN(10)        // Get as int with default

// Boolean conversions
b := value.Bool()             // Get as bool
bDef := value.BoolN(false)    // Get as bool with default

// Float conversions
f := value.Float64()          // Get as float64
fDef := value.Float64N(1.0)   // Get as float64 with default

// Time conversions
t := value.Time("2006-01-02") // Parse as time.Time with layout
tDef := value.TimeN(time.Now(), "2006-01-02") // With default value

// Special formats
csv := value.CSV()            // Parse as comma-separated values
base64 := value.FromBase64()  // Decode from base64
```

### Request Body Handling

#### `Clone(r *http.Request) *http.Request`

Clones an HTTP request while preserving body content.

```go
clonedRequest := request.Clone(req)
```

#### `Read(r *http.Request) ([]byte, error)`

Reads request body while preserving it for subsequent reads.

```go
// Read body without consuming it
body, err := request.Read(r)
if err != nil {
    return err
}

// Body can still be read again by other handlers/middleware
```

## Type-Safe Value Handling

The Value type provides a fluent API for safely handling request parameters:

```go
// Extracting and converting a parameter in one line
userID := request.PathValue(r, "id").Int64()

// Handling optional parameters with defaults
page := request.QueryValue(r, "page").IntN(1)
pageSize := request.QueryValue(r, "pageSize").IntN(10)

// Parsing CSV parameters
tags := request.QueryValue(r, "tags").CSV()

// Parsing dates
startDate := request.QueryValue(r, "startDate").Time("2006-01-02")

// Chaining operations
token := request.HeaderValue(r, "Authorization").
    Split(" "). // Split "Bearer token123"
    Get(1).     // Get the second part
    String()    // Convert to string
```

## Validation Integration

When using `DecodeJSON`, if your request type implements a `Validate() error` method, it will be automatically called:

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
    errs := cause.Map{}
    
    if r.Name == "" {
        errs["name"] = cause.Required("name", r.Name)
    }
    
    if r.Email == "" {
        errs["email"] = cause.Required("email", r.Email)
    } else if !strings.Contains(r.Email, "@") {
        errs["email"] = cause.Invalid("email", r.Email)
    }
    
    return errs.Err() // Returns nil if no errors
}
```

## Request Reading

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Read the body without consuming it
        body, err := request.Read(r)
        if err != nil {
            http.Error(w, "Failed to read request", http.StatusInternalServerError)
            return
        }
        
        // Log the request body
        log.Printf("Request Body: %s", body)
        
        // Pass the request to the next handler
        // The body is still available for reading
        next.ServeHTTP(w, r)
    })
}
```

## Best Practices

1. **Validation**: Always validate request data before processing
2. **Default Values**: Use `*N` methods for providing sensible defaults
3. **Error Handling**: Properly handle conversion errors for user input
4. **Body Reading**: Use `Read()` over manual body reading to preserve request body
5. **Security**: Validate and sanitize all user input before use

## Integration with Response Package

The request package works seamlessly with the response package:

```go
func Handler(w http.ResponseWriter, r *http.Request) {
    var req UserRequest
    
    // Parse and validate request
    if err := request.DecodeJSON(r, &req); err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    // Process request...
    
    response.OK(w, result)
}
```
