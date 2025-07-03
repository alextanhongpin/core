// Package pagination provides cursor-based and offset-based pagination utilities for HTTP APIs.
//
// This package implements both modern cursor-based pagination (recommended for large datasets)
// and traditional offset-based pagination (useful for simple use cases with page numbers).
//
// Cursor-based pagination is more efficient and consistent for large datasets as it:
// - Avoids the "page drift" problem when data is added/removed during pagination
// - Provides stable pagination even when the underlying data changes
// - Scales better for large datasets by using indexed fields for navigation
//
// Example usage:
//
//	// Cursor-based pagination for a user list API
//	cursor := pagination.NewCursor[int](10) // 10 items per page
//	cursor.After = 100 // Start after user ID 100
//
//	users, pagination, err := userService.List(ctx, cursor)
//	if err != nil {
//		return err
//	}
//
//	// Offset-based pagination for simpler use cases
//	offset := &pagination.OffsetPagination{Limit: 10, Offset: 0}
//	users, pageInfo, err := userService.ListWithOffset(ctx, offset)
//
// The package also provides utilities for encoding/decoding cursors to/from
// base64 strings for safe transmission in HTTP APIs.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// CursorType represents the type of cursor field used for pagination.
//
// This enumeration helps identify the data type of cursor values, enabling
// proper encoding, decoding, and validation of cursor parameters.
type CursorType string

const (
	// CursorTypeString indicates the cursor field is a string value
	CursorTypeString CursorType = "string"
	// CursorTypeInt indicates the cursor field is an integer value
	CursorTypeInt CursorType = "int"
	// CursorTypeTime indicates the cursor field is a timestamp value
	CursorTypeTime CursorType = "time"
)

// Cursor represents cursor-based pagination parameters for forward and backward navigation.
//
// Cursor-based pagination uses opaque tokens (cursors) to navigate through result sets.
// It supports both forward pagination (using After/First) and backward pagination
// (using Before/Last), providing efficient navigation through large datasets.
//
// Type parameter T represents the type of the cursor field (e.g., int for ID-based
// cursors, time.Time for timestamp-based cursors, or string for composite cursors).
type Cursor[T any] struct {
	// After specifies the cursor position to start after for forward pagination
	After T `json:"after"`
	// Before specifies the cursor position to end before for backward pagination
	Before T `json:"before,omitempty"`
	// First specifies the number of items to return in forward pagination
	First int `json:"first"`
	// Last specifies the number of items to return in backward pagination
	Last int `json:"last,omitempty"`
}

// NewCursor creates a new cursor with default values for forward pagination.
//
// This constructor sets up a cursor for forward pagination with the specified
// page size. The After field will be the zero value of type T, and Before/Last
// will remain unset.
//
// Parameters:
//   - first: The number of items to return per page
//
// Returns:
//   - A new cursor configured for forward pagination
//
// Example:
//
//	// Create cursor for 20 items per page
//	cursor := pagination.NewCursor[int](20)
//	cursor.After = 100 // Start after ID 100
func NewCursor[T any](first int) *Cursor[T] {
	return &Cursor[T]{
		First: first,
	}
}

// Limit converts the First/Last into database limit, and fetches an additional row
// to check if there are more items.
//
// This method adds 1 to the requested limit to enable detection of whether
// additional pages are available. The extra row is used to determine pagination
// state but is not included in the final result set.
//
// Returns:
//   - The database limit (requested size + 1 for pagination detection)
//
// Example:
//
//	cursor := &Cursor[int]{First: 10}
//	dbLimit := cursor.Limit() // Returns 11
//	// Query the database with limit 11, return 10 items, use 11th to detect hasNext
func (c *Cursor[T]) Limit() int {
	if c.Last > 0 {
		return c.Last + 1
	}
	return c.First + 1
}

// IsForward returns true if this is forward pagination (using First).
//
// Forward pagination moves through results in ascending order of the cursor field,
// typically used for "next page" navigation.
//
// Returns:
//   - true if this cursor is configured for forward pagination
func (c *Cursor[T]) IsForward() bool {
	return c.First > 0
}

// IsBackward returns true if this is backward pagination (using Last).
//
// Backward pagination moves through results in descending order of the cursor field,
// typically used for "previous page" navigation.
//
// Returns:
//   - true if this cursor is configured for backward pagination
func (c *Cursor[T]) IsBackward() bool {
	return c.Last > 0
}

// Validate validates cursor parameters against business rules.
//
// This method ensures that cursor parameters are valid and within acceptable
// limits to prevent abuse and maintain system performance.
//
// Validation rules:
// - First and Last must be non-negative
// - Cannot specify both First and Last simultaneously
// - Must specify either First or Last (not neither)
// - Pagination limit cannot exceed the specified maximum
//
// Parameters:
//   - maxLimit: The maximum allowed page size
//
// Returns:
//   - An error if validation fails, nil if parameters are valid
//
// Example:
//
//	if err := cursor.Validate(100); err != nil {
//		return fmt.Errorf("invalid pagination parameters: %w", err)
//	}
func (c *Cursor[T]) Validate(maxLimit int) error {
	if c.First < 0 || c.Last < 0 {
		return fmt.Errorf("pagination limits must be non-negative")
	}

	if c.First > 0 && c.Last > 0 {
		return fmt.Errorf("cannot specify both first and last")
	}

	if c.First == 0 && c.Last == 0 {
		return fmt.Errorf("must specify either first or last")
	}

	if c.First > maxLimit || c.Last > maxLimit {
		return fmt.Errorf("pagination limit cannot exceed %d", maxLimit)
	}

	return nil
}

// Pagination represents paginated results with metadata for navigation.
//
// This structure contains the actual result items along with pagination
// metadata that clients can use to navigate through the result set.
// It supports both cursor-based and offset-based pagination patterns.
type Pagination[T any] struct {
	// Items contains the actual result data for this page
	Items []T `json:"items"`
	// Cursor contains the cursor used to generate this page (optional)
	Cursor *Cursor[T] `json:"cursor,omitempty"`
	// HasNext indicates if there are more items after this page
	HasNext bool `json:"hasNext"`
	// HasPrev indicates if there are items before this page
	HasPrev bool `json:"hasPrev"`
	// TotalCount contains the total number of items across all pages (optional, expensive to compute)
	TotalCount *int64 `json:"totalCount,omitempty"`
	// PageInfo contains detailed pagination information for GraphQL-style APIs
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

// PageInfo contains detailed pagination information following GraphQL Relay specification.
//
// This structure provides comprehensive pagination metadata that clients can use
// to build navigation UIs and understand the current position within the result set.
type PageInfo struct {
	// HasPrevPage indicates if there are items before the current page
	HasPrevPage bool `json:"hasPrevPage"`
	// HasNextPage indicates if there are items after the current page
	HasNextPage bool `json:"hasNextPage"`
	// StartCursor is the cursor of the first item in the current page
	StartCursor string `json:"startCursor,omitempty"`
	// EndCursor is the cursor of the last item in the current page
	EndCursor string `json:"endCursor,omitempty"`
	// TotalCount is the total number of items across all pages (optional)
	TotalCount *int64 `json:"totalCount,omitempty"`
	// PageSize is the requested page size
	PageSize int `json:"pageSize"`
	// CurrentPage is the current page number (for offset-based pagination)
	CurrentPage *int `json:"currentPage,omitempty"`
	// TotalPages is the total number of pages (for offset-based pagination)
	TotalPages *int `json:"totalPages,omitempty"`
}

// OffsetPagination represents offset-based pagination parameters.
//
// Offset-based pagination uses limit/offset parameters to navigate through
// result sets. While simpler to understand, it can suffer from consistency
// issues when data is modified during pagination ("page drift").
//
// This approach is suitable for:
// - Small to medium datasets where consistency isn't critical
// - UIs that need to show page numbers or jump to specific pages
// - Cases where the total count is important for the user experience
type OffsetPagination struct {
	// Limit specifies the maximum number of items to return
	Limit int `json:"limit"`
	// Offset specifies the number of items to skip
	Offset int `json:"offset"`
	// Total contains the total number of items across all pages
	Total int64 `json:"total"`
	// Page represents the current page number (1-based)
	Page int `json:"page"`
}

// NewOffsetPagination creates offset pagination from page and limit
func NewOffsetPagination(page, limit int) *OffsetPagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	return &OffsetPagination{
		Limit:  limit,
		Offset: offset,
		Page:   page,
	}
}

// Validate validates offset pagination parameters
func (op *OffsetPagination) Validate(maxLimit int) error {
	if op.Limit < 1 {
		return fmt.Errorf("limit must be at least 1")
	}

	if op.Limit > maxLimit {
		return fmt.Errorf("limit cannot exceed %d", maxLimit)
	}

	if op.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}

	if op.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}

	return nil
}

// TotalPages calculates the total number of pages
func (op *OffsetPagination) TotalPages() int {
	if op.Limit == 0 {
		return 0
	}
	return int((op.Total + int64(op.Limit) - 1) / int64(op.Limit))
}

// HasNext returns true if there are more pages
func (op *OffsetPagination) HasNext() bool {
	return op.Page < op.TotalPages()
}

// HasPrev returns true if there are previous pages
func (op *OffsetPagination) HasPrev() bool {
	return op.Page > 1
}

// OffsetPaginatedResult represents offset-paginated results
type OffsetPaginatedResult[T any] struct {
	Items      []T               `json:"items"`
	Pagination *OffsetPagination `json:"pagination"`
}

// Paginate performs cursor-based pagination on a slice
func Paginate[T any](items []T, cursor *Cursor[T]) *Pagination[T] {
	if cursor.IsBackward() {
		return paginateBackward(items, cursor)
	}
	return paginateForward(items, cursor)
}

// paginateForward handles forward pagination
func paginateForward[T any](items []T, cursor *Cursor[T]) *Pagination[T] {
	hasNext := len(items) > cursor.First
	if hasNext {
		items = items[:cursor.First]
	}

	result := &Pagination[T]{
		Items:   items,
		HasNext: hasNext,
		HasPrev: false, // We'd need additional context to determine this
	}

	if len(items) > 0 {
		result.Cursor = &Cursor[T]{
			After: items[len(items)-1],
			First: cursor.First,
		}
	} else {
		result.Cursor = &Cursor[T]{
			First: cursor.First,
		}
	}

	return result
}

// paginateBackward handles backward pagination
func paginateBackward[T any](items []T, cursor *Cursor[T]) *Pagination[T] {
	hasPrev := len(items) > cursor.Last
	if hasPrev {
		items = items[1:] // Remove the first item (it's the indicator)
	}

	// Reverse the items for backward pagination
	for i := len(items)/2 - 1; i >= 0; i-- {
		opp := len(items) - 1 - i
		items[i], items[opp] = items[opp], items[i]
	}

	result := &Pagination[T]{
		Items:   items,
		HasNext: false, // We'd need additional context to determine this
		HasPrev: hasPrev,
	}

	if len(items) > 0 {
		result.Cursor = &Cursor[T]{
			Before: items[0],
			Last:   cursor.Last,
		}
	} else {
		result.Cursor = &Cursor[T]{
			Last: cursor.Last,
		}
	}

	return result
}

// PaginateWithTotal adds total count to pagination
func PaginateWithTotal[T any](items []T, cursor *Cursor[T], total int64) *Pagination[T] {
	result := Paginate(items, cursor)
	result.TotalCount = &total
	return result
}

// PaginateOffset performs offset-based pagination
func PaginateOffset[T any](items []T, pagination *OffsetPagination) *OffsetPaginatedResult[T] {
	pagination.Total = int64(len(items))

	start := pagination.Offset
	end := start + pagination.Limit

	if start >= len(items) {
		return &OffsetPaginatedResult[T]{
			Items:      []T{},
			Pagination: pagination,
		}
	}

	if end > len(items) {
		end = len(items)
	}

	return &OffsetPaginatedResult[T]{
		Items:      items[start:end],
		Pagination: pagination,
	}
}

// EncodeCursor encodes a cursor value to a base64 string
func EncodeCursor(value any) string {
	if value == nil {
		return ""
	}

	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 cursor string to a value
func DecodeCursor[T any](cursor string) (T, error) {
	var zero T

	if cursor == "" {
		return zero, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return zero, fmt.Errorf("invalid cursor format: %w", err)
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("invalid cursor data: %w", err)
	}

	return value, nil
}

// CursorFromString creates a cursor from string values (for URL parameters)
func CursorFromString(after, before, first, last string) (*Cursor[string], error) {
	cursor := &Cursor[string]{
		After:  after,
		Before: before,
	}

	if first != "" {
		f, err := strconv.Atoi(first)
		if err != nil {
			return nil, fmt.Errorf("invalid first parameter: %w", err)
		}
		cursor.First = f
	}

	if last != "" {
		l, err := strconv.Atoi(last)
		if err != nil {
			return nil, fmt.Errorf("invalid last parameter: %w", err)
		}
		cursor.Last = l
	}

	return cursor, nil
}

// TimeCursor represents time-based cursor pagination
type TimeCursor struct {
	After  *time.Time `json:"after,omitempty"`
	Before *time.Time `json:"before,omitempty"`
	First  int        `json:"first"`
	Last   int        `json:"last,omitempty"`
}

// NewTimeCursor creates a time-based cursor
func NewTimeCursor(first int) *TimeCursor {
	return &TimeCursor{
		First: first,
	}
}

// WithAfter sets the after time
func (tc *TimeCursor) WithAfter(t time.Time) *TimeCursor {
	tc.After = &t
	return tc
}

// WithBefore sets the before time
func (tc *TimeCursor) WithBefore(t time.Time) *TimeCursor {
	tc.Before = &t
	return tc
}

// Limit returns the query limit
func (tc *TimeCursor) Limit() int {
	if tc.Last > 0 {
		return tc.Last + 1
	}
	return tc.First + 1
}

// IsForward returns true if this is forward pagination
func (tc *TimeCursor) IsForward() bool {
	return tc.First > 0
}

// IsBackward returns true if this is backward pagination
func (tc *TimeCursor) IsBackward() bool {
	return tc.Last > 0
}

// BuildPageInfo creates PageInfo from pagination results
func BuildPageInfo[T any](pagination *Pagination[T], encodeCursor func(T) string) *PageInfo {
	pageInfo := &PageInfo{
		HasPrevPage: pagination.HasPrev,
		HasNextPage: pagination.HasNext,
		PageSize:    len(pagination.Items),
	}

	if pagination.TotalCount != nil {
		pageInfo.TotalCount = pagination.TotalCount
	}

	if len(pagination.Items) > 0 {
		pageInfo.StartCursor = encodeCursor(pagination.Items[0])
		pageInfo.EndCursor = encodeCursor(pagination.Items[len(pagination.Items)-1])
	}

	return pageInfo
}
