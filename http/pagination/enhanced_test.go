package pagination_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorPagination(t *testing.T) {
	t.Run("forward pagination", func(t *testing.T) {
		items := makeInts(11) // 1-11
		cursor := pagination.NewCursor[int](10)

		result := pagination.Paginate(items, cursor)

		assert.True(t, result.HasNext)
		assert.False(t, result.HasPrev)
		assert.Len(t, result.Items, 10)
		assert.Equal(t, 10, result.Cursor.After)
		assert.Equal(t, 10, result.Cursor.First)
	})

	t.Run("forward pagination no more items", func(t *testing.T) {
		items := makeInts(5) // 1-5
		cursor := pagination.NewCursor[int](10)

		result := pagination.Paginate(items, cursor)

		assert.False(t, result.HasNext)
		assert.Len(t, result.Items, 5)
	})

	t.Run("backward pagination", func(t *testing.T) {
		items := makeInts(11) // Should return 10 items
		cursor := &pagination.Cursor[int]{
			Last: 10,
		}

		result := pagination.Paginate(items, cursor)

		assert.True(t, result.HasPrev)
		assert.False(t, result.HasNext)
		assert.Len(t, result.Items, 10)
	})

	t.Run("pagination with total count", func(t *testing.T) {
		items := makeInts(10)
		cursor := pagination.NewCursor[int](10)
		totalCount := int64(100)

		result := pagination.PaginateWithTotal(items, cursor, totalCount)

		assert.NotNil(t, result.TotalCount)
		assert.Equal(t, int64(100), *result.TotalCount)
	})
}

func TestCursorValidation(t *testing.T) {
	tests := []struct {
		name      string
		cursor    *pagination.Cursor[int]
		maxLimit  int
		wantError bool
	}{
		{
			name:      "valid forward cursor",
			cursor:    &pagination.Cursor[int]{First: 10},
			maxLimit:  100,
			wantError: false,
		},
		{
			name:      "valid backward cursor",
			cursor:    &pagination.Cursor[int]{Last: 10},
			maxLimit:  100,
			wantError: false,
		},
		{
			name:      "negative first",
			cursor:    &pagination.Cursor[int]{First: -1},
			maxLimit:  100,
			wantError: true,
		},
		{
			name:      "both first and last",
			cursor:    &pagination.Cursor[int]{First: 10, Last: 10},
			maxLimit:  100,
			wantError: true,
		},
		{
			name:      "neither first nor last",
			cursor:    &pagination.Cursor[int]{},
			maxLimit:  100,
			wantError: true,
		},
		{
			name:      "exceeds max limit",
			cursor:    &pagination.Cursor[int]{First: 150},
			maxLimit:  100,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cursor.Validate(tt.maxLimit)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOffsetPagination(t *testing.T) {
	t.Run("create offset pagination", func(t *testing.T) {
		pagination := pagination.NewOffsetPagination(2, 10)

		assert.Equal(t, 2, pagination.Page)
		assert.Equal(t, 10, pagination.Limit)
		assert.Equal(t, 10, pagination.Offset) // (2-1) * 10
	})

	t.Run("paginate with offset", func(t *testing.T) {
		items := makeInts(25) // 1-25
		paginator := pagination.NewOffsetPagination(2, 10)

		result := pagination.PaginateOffset(items, paginator)

		assert.Equal(t, int64(25), result.Pagination.Total)
		assert.Len(t, result.Items, 10)
		assert.Equal(t, 11, result.Items[0]) // Second page starts at 11
		assert.Equal(t, 20, result.Items[9]) // Second page ends at 20
	})

	t.Run("pagination info", func(t *testing.T) {
		paginator := &pagination.OffsetPagination{
			Page:  2,
			Limit: 10,
			Total: 25,
		}

		assert.Equal(t, 3, paginator.TotalPages()) // ceil(25/10) = 3
		assert.True(t, paginator.HasNext())        // Page 2 < 3
		assert.True(t, paginator.HasPrev())        // Page 2 > 1

		paginator.Page = 1
		assert.False(t, paginator.HasPrev()) // Page 1 = 1

		paginator.Page = 3
		assert.False(t, paginator.HasNext()) // Page 3 = 3
	})

	t.Run("offset validation", func(t *testing.T) {
		tests := []struct {
			name       string
			pagination *pagination.OffsetPagination
			maxLimit   int
			wantError  bool
		}{
			{
				name:       "valid pagination",
				pagination: &pagination.OffsetPagination{Page: 1, Limit: 10, Offset: 0},
				maxLimit:   100,
				wantError:  false,
			},
			{
				name:       "zero limit",
				pagination: &pagination.OffsetPagination{Page: 1, Limit: 0, Offset: 0},
				maxLimit:   100,
				wantError:  true,
			},
			{
				name:       "exceeds max limit",
				pagination: &pagination.OffsetPagination{Page: 1, Limit: 150, Offset: 0},
				maxLimit:   100,
				wantError:  true,
			},
			{
				name:       "negative offset",
				pagination: &pagination.OffsetPagination{Page: 1, Limit: 10, Offset: -1},
				maxLimit:   100,
				wantError:  true,
			},
			{
				name:       "zero page",
				pagination: &pagination.OffsetPagination{Page: 0, Limit: 10, Offset: 0},
				maxLimit:   100,
				wantError:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.pagination.Validate(tt.maxLimit)
				if tt.wantError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestCursorEncoding(t *testing.T) {
	t.Run("encode and decode string cursor", func(t *testing.T) {
		original := "user_123"
		encoded := pagination.EncodeCursor(original)

		assert.NotEmpty(t, encoded)

		decoded, err := pagination.DecodeCursor[string](encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("encode and decode int cursor", func(t *testing.T) {
		original := 12345
		encoded := pagination.EncodeCursor(original)

		decoded, err := pagination.DecodeCursor[int](encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("encode and decode struct cursor", func(t *testing.T) {
		type UserCursor struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		original := UserCursor{ID: 123, Name: "John"}
		encoded := pagination.EncodeCursor(original)

		decoded, err := pagination.DecodeCursor[UserCursor](encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("decode invalid cursor", func(t *testing.T) {
		_, err := pagination.DecodeCursor[string]("invalid-base64!")
		assert.Error(t, err)
	})

	t.Run("encode nil cursor", func(t *testing.T) {
		encoded := pagination.EncodeCursor(nil)
		assert.Empty(t, encoded)
	})
}

func TestCursorFromString(t *testing.T) {
	t.Run("valid cursor from strings", func(t *testing.T) {
		cursor, err := pagination.CursorFromString("after_value", "", "10", "")

		require.NoError(t, err)
		assert.Equal(t, "after_value", cursor.After)
		assert.Equal(t, "", cursor.Before)
		assert.Equal(t, 10, cursor.First)
		assert.Equal(t, 0, cursor.Last)
	})

	t.Run("invalid first parameter", func(t *testing.T) {
		_, err := pagination.CursorFromString("", "", "invalid", "")
		assert.Error(t, err)
	})

	t.Run("invalid last parameter", func(t *testing.T) {
		_, err := pagination.CursorFromString("", "", "", "invalid")
		assert.Error(t, err)
	})
}

func TestTimeCursor(t *testing.T) {
	t.Run("create time cursor", func(t *testing.T) {
		now := time.Now()
		cursor := pagination.NewTimeCursor(10).
			WithAfter(now).
			WithBefore(now.Add(time.Hour))

		assert.Equal(t, 10, cursor.First)
		assert.Equal(t, now, *cursor.After)
		assert.Equal(t, now.Add(time.Hour), *cursor.Before)
		assert.True(t, cursor.IsForward())
		assert.False(t, cursor.IsBackward())
	})

	t.Run("backward time cursor", func(t *testing.T) {
		cursor := &pagination.TimeCursor{
			Last: 5,
		}

		assert.False(t, cursor.IsForward())
		assert.True(t, cursor.IsBackward())
		assert.Equal(t, 6, cursor.Limit()) // Last + 1
	})
}

func TestBuildPageInfo(t *testing.T) {
	t.Run("build page info from pagination", func(t *testing.T) {
		items := makeInts(10)
		cursor := pagination.NewCursor[int](10)
		paginationResult := pagination.Paginate(items, cursor)

		// Mock encoder function
		encodeCursor := func(item int) string {
			return pagination.EncodeCursor(item)
		}

		pageInfo := pagination.BuildPageInfo(paginationResult, encodeCursor)

		assert.Equal(t, 10, pageInfo.PageSize)
		assert.NotEmpty(t, pageInfo.StartCursor)
		assert.NotEmpty(t, pageInfo.EndCursor)
	})

	t.Run("build page info with total count", func(t *testing.T) {
		items := makeInts(10)
		cursor := pagination.NewCursor[int](10)
		totalCount := int64(100)
		paginationResult := pagination.PaginateWithTotal(items, cursor, totalCount)

		encodeCursor := func(item int) string {
			return pagination.EncodeCursor(item)
		}

		pageInfo := pagination.BuildPageInfo(paginationResult, encodeCursor)

		assert.NotNil(t, pageInfo.TotalCount)
		assert.Equal(t, int64(100), *pageInfo.TotalCount)
	})

	t.Run("build page info from empty pagination", func(t *testing.T) {
		emptyPagination := &pagination.Pagination[int]{
			Items:   []int{},
			HasNext: false,
			HasPrev: false,
		}

		encodeCursor := func(item int) string {
			return pagination.EncodeCursor(item)
		}

		pageInfo := pagination.BuildPageInfo(emptyPagination, encodeCursor)

		assert.Equal(t, 0, pageInfo.PageSize)
		assert.Empty(t, pageInfo.StartCursor)
		assert.Empty(t, pageInfo.EndCursor)
	})
}
