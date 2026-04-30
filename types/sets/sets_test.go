package sets_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/sets"
	"github.com/go-openapi/testify/assert"
)

func TestSet(t *testing.T) {

	t.Run("two equal sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.Of(1, 2, 3)
		s2 := sets.Of(1, 2, 3)

		assert.Equal([]int{1, 2, 3}, s1.Union(s2).All())
		assert.Equal([]int{1, 2, 3}, s1.Intersect(s2).All())
		assert.Equal([]int(nil), s1.Difference(s2).All())
		assert.True(s1.Union(s2).Equal(s1))
		assert.True(s1.Equal(s2))
		assert.True(s1.Intersect(s2).Equal(s1))
		assert.True(s1.Difference(s2).Equal(sets.Of[int]()))
		assert.Equal(3, s1.Len())
		assert.Equal([]int{1, 2, 3}, s1.All())
		assert.True(s1.Has(1))
		assert.True(s1.Has(2))
		assert.True(s1.Has(3))
		assert.True(!s1.Has(4))
	})

	t.Run("two partial overlapping sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.Of(1, 2, 3)
		s2 := sets.Of(2, 3, 4)

		assert.Equal([]int{1, 2, 3, 4}, s1.Union(s2).All())
		assert.Equal([]int{2, 3}, s1.Intersect(s2).All())
		assert.Equal([]int{1}, s1.Difference(s2).All())
	})

	t.Run("two non-overlapping sets", func(t *testing.T) {
		assert := assert.New(t)

		s1 := sets.Of(1, 2, 3)
		s2 := sets.Of(4, 5, 6)

		assert.Equal([]int{1, 2, 3, 4, 5, 6}, s1.Union(s2).All())
		assert.Equal([]int(nil), s1.Intersect(s2).All())
		assert.Equal([]int{1, 2, 3}, s1.Difference(s2).All())
	})

	t.Run("empty set", func(t *testing.T) {
		is := assert.New(t)
		is.True(sets.New[int]().IsEmpty())
		is.True(sets.Of[int]().IsEmpty())
		is.True((&sets.Set[int]{}).IsEmpty())
	})

	t.Run("unique", func(t *testing.T) {
		s := sets.Of(1, 2, 1, 2)
		is := assert.New(t)
		is.Equal(2, s.Len())
	})
}
