package sliceutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := sliceutil.Find(n, func(i int) bool {
			return n[i] == 3
		})
		assert.True(ok)
		assert.Equal(3, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := sliceutil.Find(n, func(i int) bool {
			return n[i] == 3
		})
		assert.False(ok)
		assert.Equal(0, m)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := sliceutil.Find(n, func(i int) bool {
			return n[i] == 99
		})
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestFilter(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		oddn := sliceutil.Filter(n, func(i int) bool {
			return n[i]%2 == 1
		})

		assert.Equal([]int{1, 3, 5}, oddn)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		oddn := sliceutil.Filter(n, func(i int) bool {
			return n[i]%2 == 1
		})

		assert.Equal([]int{}, oddn)
	})
}

func TestHead(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := sliceutil.Head(n)
		assert.True(ok)
		assert.Equal(1, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := sliceutil.Head(n)
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestTail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		m, ok := sliceutil.Tail(n)
		assert.True(ok)
		assert.Equal(5, m)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		m, ok := sliceutil.Tail(n)
		assert.False(ok)
		assert.Equal(0, m)
	})
}

func TestTake(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		assert.Equal([]int{}, sliceutil.Take(n, 0))
		assert.Equal([]int{1}, sliceutil.Take(n, 1))
		assert.Equal([]int{1, 2, 3, 4, 5}, sliceutil.Take(n, 5))
		assert.Equal(n, sliceutil.Take(n, 10))
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		assert.Equal(n, sliceutil.Take(n, 0))
		assert.Equal(n, sliceutil.Take(n, 1))
		assert.Equal(n, sliceutil.Take(n, 5))
		assert.Equal(n, sliceutil.Take(n, 10))
	})
}
