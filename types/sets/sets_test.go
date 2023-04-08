package sets_test

import (
	"testing"

	"github.com/alextanhongpin/go-core-microservice/types/sets"
	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {

	t.Run("two equal sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.New(1, 2, 3)
		s2 := sets.New(1, 2, 3)

		assert.Equal([]int{1, 2, 3}, s1.Union(s2).All())
		assert.Equal([]int{1, 2, 3}, s1.Intersect(s2).All())
		assert.Equal([]int{}, s1.Difference(s2).All())
		assert.True(s1.Union(s2).Equal(s1))
		assert.True(s1.Equal(s2))
		assert.True(s1.Intersect(s2).Equal(s1))
		assert.True(s1.Difference(s2).Equal(sets.New[int]()))
		assert.Equal(3, s1.Len())
		assert.Equal([]int{1, 2, 3}, s1.All())
		assert.True(s1.Has(1))
		assert.True(s1.Has(2))
		assert.True(s1.Has(3))
		assert.True(!s1.Has(4))
	})

	t.Run("two partial overlapping sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.New(1, 2, 3)
		s2 := sets.New(2, 3, 4)

		assert.Equal([]int{1, 2, 3, 4}, s1.Union(s2).All())
		assert.Equal([]int{2, 3}, s1.Intersect(s2).All())
		assert.Equal([]int{1}, s1.Difference(s2).All())
	})

	t.Run("two non-overlapping sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.New(1, 2, 3)
		s2 := sets.New(4, 5, 6)

		assert.Equal([]int{1, 2, 3, 4, 5, 6}, s1.Union(s2).All())
		assert.Equal([]int{}, s1.Intersect(s2).All())
		assert.Equal([]int{1, 2, 3}, s1.Difference(s2).All())
	})
}
