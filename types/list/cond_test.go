package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

func TestCond(t *testing.T) {
	n := []int{1, 2, 3, 4, 5, 6}

	lessThan := func(m int) func(int) bool {
		return func(val int) bool {
			return val < m
		}
	}

	t.Run("all", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(list.All(n, lessThan(10)))
		assert.False(list.All(n, lessThan(-10)))
		assert.False(list.All([]int{}, lessThan(5)))
	})

	t.Run("any", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(list.Any(n, lessThan(10)))
		assert.False(list.Any(n, lessThan(-10)))
		assert.False(list.Any([]int{}, lessThan(5)))
	})

	t.Run("some", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(list.Some(n, lessThan(10)))
		assert.False(list.Some(n, lessThan(-10)))
		assert.False(list.Some([]int{}, lessThan(5)))
	})

	t.Run("none", func(t *testing.T) {
		assert := assert.New(t)
		assert.False(list.None(n, lessThan(10)))
		assert.True(list.None(n, lessThan(-10)))
		assert.True(list.None([]int{}, lessThan(5)))
	})
}
