package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

func TestMath(t *testing.T) {
	n := []int{1, 2, 3, 4, 5}
	m := []int{-1, -2, -3, -4, -5}

	t.Run("Sum", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(15, list.Sum(n))
		assert.Equal(0, list.Sum([]int{}))
		assert.Equal(-15, list.Sum(m))
	})

	t.Run("Product", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(120, list.Product(n))
		assert.Equal(1, list.Product([]int{}))
		assert.Equal(-120, list.Product(m))
	})

	t.Run("Average", func(t *testing.T) {
		assert := assert.New(t)
		assert.InDelta(3.0, list.Average(n), 1e-9)
		assert.Panics(func() {
			list.Average([]int{})
		})
	})

	t.Run("ArgMax", func(t *testing.T) {
		assert := assert.New(t)
		idx := list.ArgMax(n)
		assert.Equal(4, idx)
		assert.Panics(func() {
			list.ArgMax([]int{})
		})
	})

	t.Run("ArgMin", func(t *testing.T) {
		assert := assert.New(t)
		idx := list.ArgMin(n)
		assert.Equal(0, idx)
		assert.Panics(func() {
			list.ArgMin([]int{})
		})
	})
}
