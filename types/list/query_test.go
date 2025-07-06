package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := list.Find(n, func(val int) bool {
			return val == 3
		})
		assert.True(ok)
		assert.Equal(3, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := list.Find(n, func(val int) bool {
			return val == 3
		})
		assert.False(ok)
		assert.Equal(0, m)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := list.Find(n, func(val int) bool {
			return val == 99
		})
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestFilter(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		oddn := list.Filter(n, func(val int) bool {
			return val%2 == 1
		})

		assert.Equal([]int{1, 3, 5}, oddn)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		oddn := list.Filter(n, func(val int) bool {
			return val%2 == 1
		})

		assert.Equal([]int{}, oddn)
	})
}

func TestHead(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := list.Head(n)
		assert.True(ok)
		assert.Equal(1, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := list.Head(n)
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestTail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := list.Tail(n)
		assert.True(ok)
		assert.Equal(5, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := list.Tail(n)
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestTake(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		assert.Equal([]int{}, list.Take(n, 0))
		assert.Equal([]int{1}, list.Take(n, 1))
		assert.Equal([]int{1, 2, 3, 4, 5}, list.Take(n, 5))
		assert.Equal(n, list.Take(n, 10))
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		assert.Equal(n, list.Take(n, 0))
		assert.Equal(n, list.Take(n, 1))
		assert.Equal(n, list.Take(n, 5))
		assert.Equal(n, list.Take(n, 10))
	})
}
