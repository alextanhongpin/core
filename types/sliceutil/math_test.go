package sliceutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestMath(t *testing.T) {
	n := []int{1, 2, 3, 4, 5}
	m := []int{-1, -2, -3, -4, -5}

	t.Run("Sum", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(15, sliceutil.Sum(n))
		assert.Equal(0, sliceutil.Sum([]int{}))
		assert.Equal(-15, sliceutil.Sum(m))
	})

	t.Run("Min", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(1, sliceutil.Min(n))
		assert.Equal(0, sliceutil.Min([]int{}))
		assert.Equal(-5, sliceutil.Min(m))
	})

	t.Run("Max", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(5, sliceutil.Max(n))
		assert.Equal(0, sliceutil.Max([]int{}))
		assert.Equal(-1, sliceutil.Max(m))
	})
}
