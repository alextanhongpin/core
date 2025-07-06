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
}
