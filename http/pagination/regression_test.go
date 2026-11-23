package pagination_test

import (
	"math"
	"testing"

	"github.com/alextanhongpin/core/http/pagination"
	"github.com/stretchr/testify/assert"
)

func TestPaginateBackwardDropsFarthestRow(t *testing.T) {
	// Regression: the row nearest the cursor (items[0]) was dropped instead
	// of the extra row, and the caller's slice was reversed in place.
	// Rows before 10, fetched DESC with LIMIT last+1.
	items := []int{9, 8, 7, 6}
	result := pagination.Paginate(items, &pagination.Cursor[int]{Last: 3})

	assert.True(t, result.HasPrev)
	assert.Equal(t, []int{7, 8, 9}, result.Items)
	assert.Equal(t, 7, result.Cursor.Before)
	assert.Equal(t, []int{9, 8, 7, 6}, items)
}

func TestPaginateForwardDoesNotAliasInput(t *testing.T) {
	items := []int{1, 2, 3}
	result := pagination.Paginate(items, &pagination.Cursor[int]{First: 2})
	result.Items[0] = 100
	assert.Equal(t, 1, items[0])
	assert.Equal(t, 2, result.Cursor.After)
}

func TestPaginateOffsetNegativeOffset(t *testing.T) {
	// Regression: a negative offset panicked with a slice bounds error.
	res := pagination.PaginateOffset([]int{1, 2, 3}, &pagination.OffsetPagination{Offset: -5, Limit: 2})
	assert.Equal(t, []int{1, 2}, res.Items)

	res = pagination.PaginateOffset([]int{1, 2, 3}, &pagination.OffsetPagination{Offset: 1, Limit: math.MaxInt})
	assert.Equal(t, []int{2, 3}, res.Items)
}

func TestNewOffsetPaginationDoesNotOverflow(t *testing.T) {
	p := pagination.NewOffsetPagination(math.MaxInt, 50)
	assert.GreaterOrEqual(t, p.Offset, 0)
}
