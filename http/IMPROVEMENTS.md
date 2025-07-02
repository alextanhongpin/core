# Enhanced Request, Response, and Pagination Packages

This document demonstrates the improved functionality in the request, response, and pagination packages, keeping them simple and focused while leveraging the existing `cause` package for error handling.

## Request Package Improvements

### Enhanced JSON Decoding with Options

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/request"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Name == "" {
        return errors.New("name is required")
    }
    if r.Age < 0 || r.Age > 150 {
        return errors.New("age must be between 0 and 150")
    }
    return nil
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    // Decode with size limits and required validation
    err := request.DecodeJSON(r, &req, 
        request.WithMaxBodySize(1024), // 1KB limit
        request.WithRequired(),        // Body is required
    )
    if err != nil {
        // Handle error - automatically integrates with response.ErrorJSON
        response.ErrorJSON(w, err)
        return
    }
    
    // Process request...
}
```

### Form and Query Parameter Binding

```go
type SearchRequest struct {
    Query    string   `form:"q"`
    Category string   `form:"category"`
    Page     int      `form:"page"`
    Limit    int      `form:"limit"`
    Tags     []string `form:"tags"`
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    var req SearchRequest
    
    // Bind query parameters to struct
    if err := request.DecodeQuery(r, &req); err != nil {
        // Handle error
        return
    }
    
    // Or bind form data
    if err := request.DecodeForm(r, &req); err != nil {
        // Handle error
        return
    }
}
```

### Enhanced Value Operations

```go
func exampleValueOperations(r *http.Request) {
    // Get and validate query parameters
    name := request.QueryValue(r, "name")
    if err := name.Required("name"); err != nil {
        // Handle required field error
    }
    
    // Length validation
    if err := name.Length(2, 50); err != nil {
        // Handle length validation error
    }
    
    // Email validation
    email := request.QueryValue(r, "email")
    if err := email.Email(); err != nil {
        // Handle email validation error
    }
    
    // Numeric operations with range validation
    age := request.QueryValue(r, "age")
    ageVal, err := age.IntRange(18, 100)
    if err != nil {
        // Handle range validation error
    }
    
    // Time parsing
    createdAt := request.QueryValue(r, "created_at")
    timestamp, err := createdAt.RFC3339()
    if err != nil {
        // Handle time parsing error
    }
    
    // CSV parsing
    tags := request.QueryValue(r, "tags")
    tagList := tags.CSV() // Splits "tag1,tag2,tag3" into []string
    
    // Pattern matching
    status := request.QueryValue(r, "status")
    if status.Match("active*") {
        // Handle active status patterns
    }
    
    // Check if value is in allowed list
    role := request.QueryValue(r, "role")
    allowedRoles := []string{"admin", "user", "guest"}
    if !role.InSlice(allowedRoles) {
        // Handle invalid role
    }
}
```

## Response Package Improvements

### Structured API Responses (Simplified)

The response package has been simplified to focus on consistent response formatting while leveraging the existing `cause` package for error handling.

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/response"
)

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user := map[string]any{
        "id":    123,
        "name":  "John Doe",
        "email": "john@example.com",
    }
    
    // Success response with metadata
    response.OK(w, user, 
        response.WithMeta("req-123", "v1.0", "production"),
    )
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    user := map[string]any{"id": 456, "name": "Jane Doe"}
    
    // 201 Created response
    response.Created(w, user)
}
```

### Simple Error Handling

```go
func handleError(w http.ResponseWriter, r *http.Request) {
    // Simple error responses
    response.BadRequest(w, "Invalid input provided")
    response.NotFound(w, "User not found")
    response.Unauthorized(w, "Authentication required")
    response.UnprocessableEntity(w, "Validation failed")
    
    // ErrorJSON automatically handles cause.Error and validation errors
    err := cause.New(codes.BadRequest, "INVALID_EMAIL", "Email format is invalid")
    response.ErrorJSON(w, err)
}
```

### Different Content Types

```go
func contentTypeExamples(w http.ResponseWriter, r *http.Request) {
    // JSON (default)
    response.JSON(w, map[string]string{"message": "hello"})
    
    // Pretty-printed JSON
    response.PrettyJSON(w, map[string]string{"message": "hello"})
    
    // Plain text
    response.Text(w, "Hello, World!")
    
    // HTML
    response.HTML(w, "<h1>Hello, World!</h1>")
    
    // No content
    response.NoContent(w)
}
```

### Security and CORS Headers

```go
func secureHandler(w http.ResponseWriter, r *http.Request) {
    // Set security headers
    response.SetSecurityHeaders(w)
    
    // Set cache headers (1 hour)
    response.SetCacheHeaders(w, 3600)
    
    // Set CORS headers
    origins := []string{"https://example.com"}
    methods := []string{"GET", "POST", "PUT", "DELETE"}
    headers := []string{"Content-Type", "Authorization"}
    response.CORS(w, origins, methods, headers)
    
    response.OK(w, map[string]string{"message": "secure response"})
}
```

## Key Design Principles

### Simplicity First
- **No Custom Error Types**: Leverage existing `cause` package for structured errors
- **Focus on Response Formatting**: Keep response package focused on HTTP response concerns
- **Consistent Interface**: All functions follow similar patterns for easy adoption

### Integration with Existing Code
- **Cause Package**: Automatic handling of `cause.Error` types with proper HTTP status mapping
- **Validation Errors**: Built-in support for validation error maps from the `cause` package
- **Standard Errors**: Graceful fallback for standard Go errors

## Pagination Package Improvements

### Cursor-Based Pagination

```go
package main

import (
    "github.com/alextanhongpin/core/http/pagination"
)

func listUsersWithCursor(w http.ResponseWriter, r *http.Request) {
    // Create cursor from query parameters
    after := request.QueryValue(r, "after").String()
    first := request.QueryValue(r, "first").IntOr(10)
    
    cursor := &pagination.Cursor[string]{
        After: after,
        First: first,
    }
    
    // Validate cursor
    if err := cursor.Validate(100); err != nil {
        response.BadRequest(w, err.Error())
        return
    }
    
    // Fetch data (example)
    users := fetchUsersAfter(after, cursor.Limit())
    
    // Create paginated response
    result := pagination.Paginate(users, cursor)
    
    // Convert to PageInfo for response
    pageInfo := pagination.BuildPageInfo(result, func(user User) string {
        return pagination.EncodeCursor(user.ID)
    })
    
    response.Paginated(w, result.Items, *pageInfo)
}
```

### Offset-Based Pagination

```go
func listUsersWithOffset(w http.ResponseWriter, r *http.Request) {
    page := request.QueryValue(r, "page").IntOr(1)
    limit := request.QueryValue(r, "limit").IntOr(10)
    
    // Create offset pagination
    paginator := pagination.NewOffsetPagination(page, limit)
    
    // Validate pagination
    if err := paginator.Validate(100); err != nil {
        response.BadRequest(w, err.Error())
        return
    }
    
    // Fetch data
    users := fetchAllUsers()
    result := pagination.PaginateOffset(users, paginator)
    
    // Create response with pagination info
    pageInfo := response.PageInfo{
        HasNextPage: result.Pagination.HasNext(),
        HasPrevPage: result.Pagination.HasPrev(),
        Total:       &result.Pagination.Total,
        Page:        &result.Pagination.Page,
        PerPage:     &result.Pagination.Limit,
    }
    
    totalPages := result.Pagination.TotalPages()
    pageInfo.TotalPages = &totalPages
    
    response.Paginated(w, result.Items, pageInfo)
}
```

### Time-Based Cursor Pagination

```go
func listEventsWithTimeCursor(w http.ResponseWriter, r *http.Request) {
    first := request.QueryValue(r, "first").IntOr(10)
    
    cursor := pagination.NewTimeCursor(first)
    
    // Parse time parameters
    if afterStr := request.QueryValue(r, "after").String(); afterStr != "" {
        after, err := time.Parse(time.RFC3339, afterStr)
        if err != nil {
            response.BadRequest(w, "Invalid after timestamp")
            return
        }
        cursor.WithAfter(after)
    }
    
    if beforeStr := request.QueryValue(r, "before").String(); beforeStr != "" {
        before, err := time.Parse(time.RFC3339, beforeStr)
        if err != nil {
            response.BadRequest(w, "Invalid before timestamp")
            return
        }
        cursor.WithBefore(before)
    }
    
    // Fetch events within time range
    events := fetchEventsInTimeRange(cursor.After, cursor.Before, cursor.Limit())
    
    // Create response
    response.OK(w, map[string]any{
        "events": events,
        "cursor": cursor,
    })
}
```

### Encoded Cursors

```go
func handleEncodedCursors(w http.ResponseWriter, r *http.Request) {
    // Decode cursor from URL parameter
    cursorStr := request.QueryValue(r, "cursor").String()
    cursor, err := pagination.DecodeCursor[UserCursor](cursorStr)
    if err != nil {
        response.BadRequest(w, "Invalid cursor format")
        return
    }
    
    // Use cursor for querying
    users := fetchUsersAfterCursor(cursor)
    
    // Encode cursor for next page
    if len(users) > 0 {
        lastUser := users[len(users)-1]
        nextCursor := pagination.EncodeCursor(UserCursor{
            ID:   lastUser.ID,
            Name: lastUser.Name,
        })
        
        response.OK(w, map[string]any{
            "users":      users,
            "nextCursor": nextCursor,
        })
    } else {
        response.OK(w, map[string]any{
            "users":      users,
            "nextCursor": nil,
        })
    }
}

type UserCursor struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
```

## Complete Example: RESTful API with All Features

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/http/request"
    "github.com/alextanhongpin/core/http/response"
    "github.com/alextanhongpin/core/http/pagination"
)

type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"createdAt"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Name == "" {
        return request.NewValidationError("name", "name is required")
    }
    if err := request.Value(r.Email).Email(); err != nil {
        return request.NewValidationError("email", "invalid email format")
    }
    return nil
}

type UserService struct {
    users []User
}

func (s *UserService) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    // Decode and validate request
    if err := request.DecodeJSON(r, &req, request.WithRequired()); err != nil {
        response.ErrorJSON(w, err)
        return
    }
    
    // Create user
    user := User{
        ID:        len(s.users) + 1,
        Name:      req.Name,
        Email:     req.Email,
        CreatedAt: time.Now(),
    }
    s.users = append(s.users, user)
    
    // Return created user
    response.Created(w, user,
        response.WithMeta(request.HeaderValue(r, "X-Request-ID").String(), "v1.0", "production"),
    )
}

func (s *UserService) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse pagination parameters
    cursor, err := pagination.CursorFromString(
        request.QueryValue(r, "after").String(),
        request.QueryValue(r, "before").String(),
        request.QueryValue(r, "first").String(),
        request.QueryValue(r, "last").String(),
    )
    if err != nil {
        response.BadRequest(w, "Invalid pagination parameters")
        return
    }
    
    // Set default limit if not provided
    if cursor.First == 0 && cursor.Last == 0 {
        cursor.First = 10
    }
    
    // Validate pagination
    if err := cursor.Validate(100); err != nil {
        response.BadRequest(w, err.Error())
        return
    }
    
    // Simulate pagination (in real app, this would be database query)
    result := pagination.Paginate(s.users, cursor)
    
    // Build page info
    pageInfo := pagination.BuildPageInfo(result, func(user User) string {
        return pagination.EncodeCursor(user.ID)
    })
    
    // Add total count
    total := int64(len(s.users))
    pageInfo.TotalCount = &total
    
    // Return paginated response
    response.Paginated(w, result.Items, *pageInfo,
        response.WithMeta(request.HeaderValue(r, "X-Request-ID").String(), "v1.0", "production"),
    )
}

func main() {
    service := &UserService{}
    
    http.HandleFunc("POST /users", service.CreateUser)
    http.HandleFunc("GET /users", service.ListUsers)
    
    http.ListenAndServe(":8080", nil)
}
```

## Key Improvements Summary

### Request Package
- ✅ **Enhanced JSON Decoding**: Size limits, required validation, custom options
- ✅ **Form/Query Binding**: Automatic struct binding with validation
- ✅ **Comprehensive Validation**: Email, length, range, pattern matching
- ✅ **Type Conversions**: Safe parsing with defaults and validation
- ✅ **Utility Methods**: CSV parsing, URL validation, time parsing

### Response Package
- ✅ **Structured Responses**: Consistent API response format with metadata
- ✅ **Enhanced Error Handling**: Typed errors with validation details
- ✅ **Multiple Content Types**: JSON, Text, HTML with proper headers
- ✅ **Security Features**: Security headers, CORS, cache control
- ✅ **Convenience Methods**: Pre-built status code responses

### Pagination Package
- ✅ **Cursor Pagination**: Forward/backward with validation
- ✅ **Offset Pagination**: Traditional page-based pagination
- ✅ **Time-Based Cursors**: Pagination based on timestamps
- ✅ **Cursor Encoding**: Safe base64 encoding for complex cursors
- ✅ **Validation**: Comprehensive parameter validation
- ✅ **Flexible Options**: Support for total counts, metadata

These improvements make the packages production-ready with comprehensive error handling, validation, and feature-rich APIs suitable for modern web applications.
