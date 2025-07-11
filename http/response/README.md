# HTTP Response Package

The HTTP Response package provides a robust and standardized way to handle HTTP responses in Go applications. It offers structured JSON responses, error handling, response recording, and utilities for building consistent APIs.

## Features

- **Structured JSON Responses**: Standardized response format with data, error, and pagination information
- **Smart Error Handling**: Automatic error type detection and appropriate HTTP status code mapping
- **Response Recording**: Capture and inspect HTTP responses for middleware and testing
- **Pagination Support**: Built-in pagination metadata for list endpoints
- **Type Safety**: Generic support and clear interfaces
- **Cross-Package Integration**: Works with `pagination`, `handler`, and `request` packages

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/response"
)

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: "123", Name: "Alice"}
    response.OK(w, user)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: "456", Name: "Bob"}
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

#### `OK(w http.ResponseWriter, v any, status ...int)`

Sends a successful JSON response.

#### `JSON(w http.ResponseWriter, v any, status ...int)`

Sends a custom JSON response.

### Error Responses

#### `ErrorJSON(w http.ResponseWriter, err error, status ...int)`

Sends a standardized error response.

### Pagination

#### `PageInfo` struct

Includes pagination metadata in responses.

## Best Practices

- Use standardized response formats for all endpoints.
- Include pagination metadata for list APIs.
- Integrate with `handler` and `pagination` for full API lifecycle.

## Related Packages

- [`pagination`](../pagination/README.md): Cursor-based pagination
- [`handler`](../handler/README.md): Base handler utilities
- [`request`](../request/README.md): Request parsing utilities

## License

MIT
