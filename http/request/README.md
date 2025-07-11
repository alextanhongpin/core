# HTTP Request Package

The HTTP Request package provides utilities for parsing, validating, and accessing HTTP request data in Go web applications, with a focus on type-safe value handling.

## Features

- **JSON Body Parsing**: Decode and validate JSON request bodies
- **Value Extraction**: Get values from URL parameters, forms, and headers
- **Type-Safe Conversion**: Fluent API for converting string values to various types
- **Validation Support**: Integrated validation for request structures
- **Request Body Reading**: Read and restore request body content
- **Default Values**: Convenient handling of missing or empty values
- **Cross-Package Integration**: Works with `handler`, `response`, and `pagination` packages

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

func (r CreateUserRequest) Validate() error {
    // Your validation logic here
    return nil
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := request.DecodeJSON(r, &req); err != nil {
        response.ErrorJSON(w, err)
        return
    }
    response.OK(w, map[string]string{"status": "success"})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    userID := request.PathValue(r, "id").Int()
    page := request.QueryValue(r, "page").IntN(1)
    since := request.QueryValue(r, "since").Time("2006-01-02")
    // ...
}
```

## API Reference

### JSON Decoding

#### `DecodeJSON(r *http.Request, v any) error`
Decodes JSON body into struct.

### Value Extraction

#### `PathValue(r *http.Request, key string) Value`
#### `QueryValue(r *http.Request, key string) Value`
#### `HeaderValue(r *http.Request, key string) Value`
Type-safe value extraction and conversion.

## Best Practices

- Always validate incoming request payloads.
- Use type-safe value extraction for robust code.
- Integrate with `handler` and `response` for full request/response lifecycle.

## Related Packages

- [`handler`](../handler/README.md): Base handler utilities
- [`response`](../response/README.md): Response formatting
- [`pagination`](../pagination/README.md): Cursor-based pagination

## License

MIT
