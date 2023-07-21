package sliceutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestCond(t *testing.T) {
	n := []int{1, 2, 3, 4, 5, 6}

	lessThan := func(m int) func(int) bool {
		return func(i int) bool {
			return n[i] < m
		}
	}

	t.Run("all", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(sliceutil.All(n, lessThan(10)))
		assert.False(sliceutil.All(n, lessThan(-10)))
		assert.False(sliceutil.All([]int{}, lessThan(5)))
	})

	t.Run("any", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(sliceutil.Any(n, lessThan(10)))
		assert.False(sliceutil.Any(n, lessThan(-10)))
		assert.False(sliceutil.Any([]int{}, lessThan(5)))
	})

	t.Run("some", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(sliceutil.Some(n, lessThan(10)))
		assert.False(sliceutil.Some(n, lessThan(-10)))
		assert.False(sliceutil.Some([]int{}, lessThan(5)))
	})

	t.Run("none", func(t *testing.T) {
		assert := assert.New(t)
		assert.False(sliceutil.None(n, lessThan(10)))
		assert.True(sliceutil.None(n, lessThan(-10)))
		assert.False(sliceutil.None([]int{}, lessThan(5)))
	})
}
