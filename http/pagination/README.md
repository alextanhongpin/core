# HTTP Pagination Package

The HTTP Pagination package provides robust cursor-based pagination for efficient handling of large datasets in Go web applications.

## Features

- **Cursor-Based Pagination**: Efficient navigation of large datasets using cursors instead of offset
- **Type-Safe Implementations**: Generic support for different cursor types (string, int, time)
- **Bidirectional Navigation**: Support for both forward and backward pagination
- **Automatic Has-Next Detection**: Smart detection of additional pages
- **Edge Case Handling**: Proper handling of empty results and boundaries
- **Pagination Metadata**: Structured page information for API responses
- **Cross-Package Integration**: Works with `response` and `request` packages

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/pagination"
    "github.com/alextanhongpin/core/http/response"
)

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
    cursor := pagination.NewCursor[string](10)
    if after := r.URL.Query().Get("after"); after != "" {
        cursor.After = after
    }
    items := []User{...} // From database
    paginated := pagination.Paginate(items, cursor)
    response.JSON(w, &response.Body{
        Data: paginated.Items,
        PageInfo: &response.PageInfo{
            HasNextPage: paginated.HasNextPage,
            EndCursor:   paginated.EndCursor,
        },
    })
}
```

## API Reference

### Cursor Types

```go
// Create string-based cursor (good for UUIDs, slugs)
cursor := pagination.NewCursor[string](10)

// Create int-based cursor
cursor := pagination.NewCursor[int](10)
```

### Paginate

```go
paginated := pagination.Paginate(items, cursor)
```

## Best Practices

- Use cursor-based pagination for scalable APIs.
- Return structured pagination metadata in all list endpoints.
- Integrate with `response` for standardized output.

## Related Packages

- [`response`](../response/README.md): Response formatting and pagination metadata
- [`request`](../request/README.md): Request parsing utilities

## License

MIT
