package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		l := list.New([]int{1, 2, 3})
		assert.Equal(t, []int{1, 2, 3}, l.ToSlice())
		assert.Equal(t, 3, l.Len())
		assert.False(t, l.IsEmpty())
	})

	t.Run("From", func(t *testing.T) {
		l := list.From(1, 2, 3)
		assert.Equal(t, []int{1, 2, 3}, l.ToSlice())
		assert.Equal(t, 3, l.Len())
		assert.False(t, l.IsEmpty())
	})

	t.Run("Empty", func(t *testing.T) {
		l := list.New([]int{})
		assert.Equal(t, []int{}, l.ToSlice())
		assert.Equal(t, 0, l.Len())
		assert.True(t, l.IsEmpty())
	})

	t.Run("Map", func(t *testing.T) {
		l := list.New([]int{1, 2, 3})
		doubled := l.Map(func(x int) int { return x * 2 })
		assert.Equal(t, []int{2, 4, 6}, doubled.ToSlice())
		// Original should be unchanged
		assert.Equal(t, []int{1, 2, 3}, l.ToSlice())
	})

	t.Run("Filter", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5})
		evens := l.Filter(func(x int) bool { return x%2 == 0 })
		assert.Equal(t, []int{2, 4}, evens.ToSlice())
	})

	t.Run("Chaining", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5})
		result := l.
			Filter(func(x int) bool { return x%2 == 0 }).
			Map(func(x int) int { return x * 2 })
		assert.Equal(t, []int{4, 8}, result.ToSlice())
	})

	t.Run("Reverse", func(t *testing.T) {
		l := list.New([]int{1, 2, 3})
		reversed := l.Reverse()
		assert.Equal(t, []int{3, 2, 1}, reversed.ToSlice())
	})

	t.Run("Chunk", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5})
		chunked := l.Chunk(2)
		expected := [][]int{{1, 2}, {3, 4}, {5}}
		assert.Equal(t, expected, chunked)
	})

	t.Run("Partition", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5})
		evens, odds := l.Partition(func(x int) bool { return x%2 == 0 })
		assert.Equal(t, []int{2, 4}, evens.ToSlice())
		assert.Equal(t, []int{1, 3, 5}, odds.ToSlice())
	})

	t.Run("Conditional methods", func(t *testing.T) {
		l := list.New([]int{2, 4, 6})
		assert.True(t, l.All(func(x int) bool { return x%2 == 0 }))
		assert.True(t, l.Any(func(x int) bool { return x > 5 }))
		assert.False(t, l.None(func(x int) bool { return x%2 == 0 }))
		assert.True(t, l.Some(func(x int) bool { return x > 5 }))
	})

	t.Run("Query methods", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5})
		
		// Find
		val, found := l.Find(func(x int) bool { return x > 3 })
		assert.True(t, found)
		assert.Equal(t, 4, val)
		
		// Head and Tail
		head, ok := l.Head()
		assert.True(t, ok)
		assert.Equal(t, 1, head)
		
		tail, ok := l.Tail()
		assert.True(t, ok)
		assert.Equal(t, 5, tail)
		
		// Take and Drop
		taken := l.Take(3)
		assert.Equal(t, []int{1, 2, 3}, taken.ToSlice())
		
		dropped := l.Drop(2)
		assert.Equal(t, []int{3, 4, 5}, dropped.ToSlice())
	})

	t.Run("Complex chaining", func(t *testing.T) {
		l := list.New([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		result := l.
			Filter(func(x int) bool { return x%2 == 0 }).
			Map(func(x int) int { return x * 2 }).
			Take(2).
			Reverse()
		assert.Equal(t, []int{8, 4}, result.ToSlice())
	})
}