package list_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

type user struct {
	ID   int
	Name string
}
type userDedup struct{ seen map[int]struct{} }

func (d *userDedup) Has(u *user) bool { _, ok := d.seen[u.ID]; return ok }
func (d *userDedup) Set(u *user)      { d.seen[u.ID] = struct{}{} }

func TestIterMethods(t *testing.T) {
	t.Run("DedupFunc", func(t *testing.T) {
		users := []*user{
			{ID: 1, Name: "A"},
			{ID: 2, Name: "B"},
			{ID: 1, Name: "C"},
			{ID: 3, Name: "D"},
			{ID: 2, Name: "E"},
		}
		dedup := &userDedup{seen: make(map[int]struct{})}
		it := list.IterFrom(users).DedupFunc(dedup)
		var gotIDs []int
		it.Each(func(u *user) { gotIDs = append(gotIDs, u.ID) })
		assert.ElementsMatch(t, []int{1, 2, 3}, gotIDs)
	})
	// Helper to create an Iter[int]
	iterOf := func(vs ...int) *list.Iter[int] {
		return list.IterOf(vs...)
	}

	t.Run("Take", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5).Take(3)
		assert.Equal(t, []int{1, 2, 3}, it.Collect())
	})

	t.Run("Skip", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5).Skip(2)
		assert.Equal(t, []int{3, 4, 5}, it.Collect())
	})

	t.Run("Any", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5)
		assert.True(t, it.Any(func(x int) bool { return x == 3 }))
		assert.False(t, it.Any(func(x int) bool { return x == 10 }))
	})

	t.Run("All", func(t *testing.T) {
		it := iterOf(2, 4, 6)
		assert.True(t, it.All(func(x int) bool { return x%2 == 0 }))
		assert.False(t, it.All(func(x int) bool { return x > 2 }))
	})

	t.Run("Find", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5)
		val, found := it.Find(func(x int) bool { return x > 3 })
		assert.True(t, found)
		assert.Equal(t, 4, val)
		_, found = it.Find(func(x int) bool { return x > 10 })
		assert.False(t, found)
	})

	t.Run("Count", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5)
		assert.Equal(t, 5, it.Count())
		it = iterOf()
		assert.Equal(t, 0, it.Count())
	})

	t.Run("Reduce", func(t *testing.T) {
		it := iterOf(1, 2, 3, 4, 5)
		sum := it.Reduce(func(acc, v int) int { return acc + v }, 0)
		assert.Equal(t, 15, sum)
		prod := iterOf(1, 2, 3, 4).Reduce(func(acc, v int) int { return acc * v }, 1)
		assert.Equal(t, 24, prod)
	})
}
