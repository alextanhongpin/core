# HTTP Pagination Package

The HTTP Pagination package provides robust cursor-based pagination for efficient handling of large datasets in Go web applications.

## Features

- **Cursor-Based Pagination**: Efficient navigation of large datasets using cursors instead of offset
- **Type-Safe Implementations**: Generic support for different cursor types (string, int, time)
- **Bidirectional Navigation**: Support for both forward and backward pagination
- **Automatic Has-Next Detection**: Smart detection of additional pages
- **Edge Case Handling**: Proper handling of empty results and boundaries
- **Pagination Metadata**: Structured page information for API responses

## Quick Start

```go
package main

import (
    "net/http"
    
    "github.com/alextanhongpin/core/http/pagination"
    "github.com/alextanhongpin/core/http/response"
)

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Create a cursor with 10 items per page
    cursor := pagination.NewCursor[string](10)
    
    // Parse cursor from query parameters
    if after := r.URL.Query().Get("after"); after != "" {
        cursor.After = after
    }
    
    // Get items from database
    // In real code, you'd use cursor.After to filter your query
    // and cursor.Limit() to limit results
    items := []User{...} // From database
    
    // Paginate results
    paginated := pagination.Paginate(items, cursor)
    
    // Send response with pagination info
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

// Create integer-based cursor (good for sequential IDs)
cursor := pagination.NewCursor[int](10)

// Create time-based cursor (good for timestamps)
cursor := pagination.NewCursor[time.Time](10)
```

### Core Functions

#### `NewCursor[T any](first int) *Cursor[T]`

Creates a new cursor with the specified page size.

```go
cursor := pagination.NewCursor[string](25) // 25 items per page
```

#### `Paginate[T any, C any](items []T, cursor *Cursor[C]) *PaginatedResult[T, C]`

Applies pagination to a slice of items.

```go
result := pagination.Paginate(users, cursor)

// Access paginated data
items := result.Items // Paginated items
hasNext := result.HasNextPage // True if more pages exist
endCursor := result.EndCursor // Cursor for the next page
```

#### `(c *Cursor[T]) Limit() int`

Returns the database limit to use (items per page + 1 for has-next detection).

```go
// In your repository
query := "SELECT * FROM users WHERE id > ? ORDER BY id LIMIT ?"
rows, err := db.Query(query, cursor.After, cursor.Limit())
```

#### `EncodeCursor(cursor any) (string, error)`

Encodes any cursor value to a Base64 string for safe URL usage.

```go
// Encode a cursor for API response
encoded, _ := pagination.EncodeCursor(lastID)
```

#### `DecodeCursor(cursor string, target any) error`

Decodes a Base64 cursor string into the target value.

```go
// Decode a cursor from API request
var lastID int
pagination.DecodeCursor(cursorParam, &lastID)
```

### Helper Methods

#### `(c *Cursor[T]) IsForward() bool`

Returns true if this is forward pagination (using After/First).

#### `(c *Cursor[T]) IsBackward() bool`

Returns true if this is backward pagination (using Before/Last).

#### `(c *Cursor[T]) Validate() error`

Validates cursor parameters for logical consistency.

## Pagination Result

The `Paginate` function returns a `PaginatedResult` with the following fields:

```go
type PaginatedResult[T any, C any] struct {
    Items       []T // The paginated items
    HasNextPage bool // True if more items exist
    HasPrevPage bool // True if previous page exists
    StartCursor C   // First item's cursor value
    EndCursor   C   // Last item's cursor value
}
```

## Cursor-Based Pagination Implementation

### Forward Pagination (Next Page)

```go
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Parse cursor from query
    after := r.URL.Query().Get("after")
    
    // Create cursor
    cursor := &pagination.Cursor[string]{
        After: after,
        First: 10,
    }
    
    // Repository query (pseudocode)
    query := "SELECT * FROM users"
    args := []any{}
    
    if cursor.After != "" {
        query += " WHERE id > ?"
        args = append(args, cursor.After)
    }
    
    query += " ORDER BY id LIMIT ?"
    args = append(args, cursor.Limit())
    
    // Execute query and get items...
    
    // Paginate and send response
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

### Backward Pagination (Previous Page)

```go
cursor := &pagination.Cursor[string]{
    Before: before,
    Last:   10,
}

// Repository query (pseudocode)
query := "SELECT * FROM users"
args := []any{}

if cursor.Before != "" {
    query += " WHERE id < ?"
    args = append(args, cursor.Before)
}

query += " ORDER BY id DESC LIMIT ?"
args = append(args, cursor.Limit())

// Note: For backward pagination, you'll need to reverse the items
```

## Time-Based Cursor Example

```go
// Time-based cursor
cursor := &pagination.Cursor[time.Time]{
    After: lastSeen,
    First: 10,
}

// SQL query using cursor
query := "SELECT * FROM events WHERE created_at > ? ORDER BY created_at LIMIT ?"
rows, err := db.Query(query, cursor.After, cursor.Limit())
```

## Integration with Response Package

The pagination package is designed to work seamlessly with the response package:

```go
paginated := pagination.Paginate(items, cursor)

response.JSON(w, &response.Body{
    Data: paginated.Items,
    PageInfo: &response.PageInfo{
        HasNextPage: paginated.HasNextPage,
        HasPrevPage: paginated.HasPrevPage,
        StartCursor: paginated.StartCursor,
        EndCursor:   paginated.EndCursor,
    },
})
```

## Best Practices

1. **Use Type-Appropriate Cursors**: Choose string, int, or time cursors based on your data
2. **Database Indexing**: Ensure cursor fields are properly indexed in your database
3. **Cursor Encoding**: Use `EncodeCursor` for safe transfer in URLs
4. **Consistent Sorting**: Always maintain consistent sort order for reliable pagination
5. **Error Handling**: Validate cursor parameters to catch potential issues early
