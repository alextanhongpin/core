package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// CursorType represents the type of cursor field
type CursorType string

const (
	CursorTypeString CursorType = "string"
	CursorTypeInt    CursorType = "int"
	CursorTypeTime   CursorType = "time"
)

// Cursor represents cursor-based pagination parameters
type Cursor[T any] struct {
	After  T   `json:"after"`
	Before T   `json:"before,omitempty"`
	First  int `json:"first"`
	Last   int `json:"last,omitempty"`
}

// NewCursor creates a new cursor with default values
func NewCursor[T any](first int) *Cursor[T] {
	return &Cursor[T]{
		First: first,
	}
}

// Limit converts the First/Last into database limit, and fetches an additional row
// to check if there are more items.
func (c *Cursor[T]) Limit() int {
	if c.Last > 0 {
		return c.Last + 1
	}
	return c.First + 1
}

// IsForward returns true if this is forward pagination
func (c *Cursor[T]) IsForward() bool {
	return c.First > 0
}

// IsBackward returns true if this is backward pagination
func (c *Cursor[T]) IsBackward() bool {
	return c.Last > 0
}

// Validate validates cursor parameters
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

// Pagination represents paginated results
type Pagination[T any] struct {
	Items      []T        `json:"items"`
	Cursor     *Cursor[T] `json:"cursor,omitempty"`
	HasNext    bool       `json:"hasNext"`
	HasPrev    bool       `json:"hasPrev"`
	TotalCount *int64     `json:"totalCount,omitempty"`
	PageInfo   *PageInfo  `json:"pageInfo,omitempty"`
}

// PageInfo contains detailed pagination information
type PageInfo struct {
	HasPrevPage bool   `json:"hasPrevPage"`
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor,omitempty"`
	EndCursor   string `json:"endCursor,omitempty"`
	TotalCount  *int64 `json:"totalCount,omitempty"`
	PageSize    int    `json:"pageSize"`
	CurrentPage *int   `json:"currentPage,omitempty"`
	TotalPages  *int   `json:"totalPages,omitempty"`
}

// OffsetPagination represents offset-based pagination
type OffsetPagination struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
	Page   int   `json:"page"`
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
