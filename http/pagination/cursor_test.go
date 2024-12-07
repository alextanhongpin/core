package pagination_test

import (
	"testing"

	"github.com/alextanhongpin/core/http/pagination"
	"github.com/stretchr/testify/assert"
)

func TestCursor(t *testing.T) {
	cursor := &pagination.Cursor[int]{
		First: 10,
	}

	is := assert.New(t)
	is.False(pagination.Paginate(makeInts(0), cursor).HasNext, "empty list")
	is.False(pagination.Paginate(makeInts(10), cursor).HasNext, "end of list")

	paginated := pagination.Paginate(makeInts(11), cursor)
	is.True(paginated.HasNext)
	is.Equal(10, paginated.Cursor.After)
	is.Equal(10, paginated.Cursor.First)
}

func makeInts(n int) []int {
	list := make([]int, n)
	for i := 0; i < n; i++ {
		list[i] = i + 1
	}
	return list
}
