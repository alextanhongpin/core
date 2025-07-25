package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	t.Run("From", func(t *testing.T) {
		l := list.From([]int{1, 2, 3})
		assert.Equal(t, []int{1, 2, 3}, l.Slice())
		assert.Equal(t, 3, l.Len())
		assert.False(t, l.IsEmpty())
	})

	t.Run("Of", func(t *testing.T) {
		l := list.Of(1, 2, 3)
		assert.Equal(t, []int{1, 2, 3}, l.Slice())
		assert.Equal(t, 3, l.Len())
		assert.False(t, l.IsEmpty())
	})

	t.Run("Empty", func(t *testing.T) {
		l := list.From([]int{})
		assert.Equal(t, []int{}, l.Slice())
		assert.Equal(t, 0, l.Len())
		assert.True(t, l.IsEmpty())
	})

	t.Run("Map", func(t *testing.T) {
		l := list.From([]int{1, 2, 3})
		doubled := l.Map(func(x int) int { return x * 2 })
		assert.Equal(t, []int{2, 4, 6}, doubled.Slice())
		// Original should be unchanged
		assert.Equal(t, []int{1, 2, 3}, l.Slice())
	})

	t.Run("Filter", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})
		evens := l.Filter(func(x int) bool { return x%2 == 0 })
		assert.Equal(t, []int{2, 4}, evens.Slice())
	})

	t.Run("Chaining", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})
		result := l.
			Filter(func(x int) bool { return x%2 == 0 }).
			Map(func(x int) int { return x * 2 })
		assert.Equal(t, []int{4, 8}, result.Slice())
	})

	t.Run("Reverse", func(t *testing.T) {
		l := list.From([]int{1, 2, 3})
		reversed := l.Reverse()
		assert.Equal(t, []int{3, 2, 1}, reversed.Slice())
	})

	t.Run("Chunk", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})
		chunked := l.Chunk(2)
		expected := [][]int{{1, 2}, {3, 4}, {5}}
		assert.Equal(t, expected, chunked)
	})

	t.Run("Partition", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})
		evens, odds := l.Partition(func(x int) bool { return x%2 == 0 })
		assert.Equal(t, []int{2, 4}, evens.Slice())
		assert.Equal(t, []int{1, 3, 5}, odds.Slice())
	})

	t.Run("Conditional methods", func(t *testing.T) {
		l := list.From([]int{2, 4, 6})
		assert.True(t, l.All(func(x int) bool { return x%2 == 0 }))
		assert.True(t, l.Any(func(x int) bool { return x > 5 }))
		assert.False(t, l.None(func(x int) bool { return x%2 == 0 }))
		assert.True(t, l.Some(func(x int) bool { return x > 5 }))
	})

	t.Run("Query methods", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})

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
		assert.Equal(t, []int{1, 2, 3}, taken.Slice())

		dropped := l.Drop(2)
		assert.Equal(t, []int{3, 4, 5}, dropped.Slice())
	})

	t.Run("Complex chaining", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		result := l.
			Filter(func(x int) bool { return x%2 == 0 }).
			Map(func(x int) int { return x * 2 }).
			Take(2).
			Reverse()
		assert.Equal(t, []int{8, 4}, result.Slice())
	})

	t.Run("List manipulation methods", func(t *testing.T) {
		l := list.From([]int{1, 2, 3})

		// Clone
		cloned := l.Clone()
		assert.Equal(t, l.Slice(), cloned.Slice())

		// Append
		appended := l.Append(4, 5)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, appended.Slice())
		assert.Equal(t, []int{1, 2, 3}, l.Slice()) // Original unchanged

		// Prepend
		prepended := l.Prepend(0, -1)
		assert.Equal(t, []int{0, -1, 1, 2, 3}, prepended.Slice())
		assert.Equal(t, []int{1, 2, 3}, l.Slice()) // Original unchanged
	})

	t.Run("Reduce", func(t *testing.T) {
		l := list.From([]int{1, 2, 3, 4, 5})
		sum := l.Reduce(0, func(acc interface{}, item int) interface{} {
			return acc.(int) + item
		})
		assert.Equal(t, 15, sum)
	})

	t.Run("FlatMap", func(t *testing.T) {
		l := list.From([]int{1, 2, 3})
		result := l.FlatMap(func(x int) []int {
			return []int{x, x * 2}
		})
		assert.Equal(t, []int{1, 2, 2, 4, 3, 6}, result.Slice())
	})
}
