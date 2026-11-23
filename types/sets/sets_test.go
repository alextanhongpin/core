package sets_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/alextanhongpin/core/types/sets"
	"github.com/go-openapi/testify/assert"
)

func TestUnique(t *testing.T) {
	is := assert.New(t)
	// Regression: Unique went through a map and lost the input order.
	for range 20 {
		is.Equal([]int{3, 1, 2}, sets.Unique([]int{3, 1, 3, 2, 1}))
	}
	is.Nil(sets.Unique[int](nil))
	is.Equal([]int{}, sets.Unique([]int{}))
}

func TestSet(t *testing.T) {
	is := assert.New(t)
	var s sets.Set[string] // zero value is usable
	is.True(s.IsEmpty())
	is.False(s.Has("a"))
	is.False(s.Remove("a"))
	is.True(s.Add("a"))
	is.False(s.Add("a"))
	is.Equal(2, s.AddMany("a", "b", "c"))
	is.Equal(3, s.Len())
	is.Equal(1, s.RemoveMany("b", "x"))
	is.Equal([]string{"a", "c"}, sets.Sorted(&s))
	is.Equal([]string{"a", "c"}, slices.Sorted(s.Values()))
	is.Equal([]string{"c"}, s.Filter(func(v string) bool { return v > "b" }).All())

	c := s.Clone()
	c.Add("z")
	is.False(s.Has("z"), "clone is independent")

	s.Clear()
	is.Equal(0, s.Len())
	is.Equal("{}", s.String())
	is.Equal("{1}", sets.Of(1).String())
}

func TestOperations(t *testing.T) {
	is := assert.New(t)
	a, b, c := sets.Of(1, 2, 3), sets.Of(2, 3, 4), sets.Of(3, 4, 5)

	is.Equal([]int{3}, sets.Sorted(sets.Intersection(a, b, c)))
	is.Equal([]int{1, 2, 3, 4, 5}, sets.Sorted(sets.Union(a, b, c)))
	is.Equal([]int{1}, sets.Sorted(sets.Difference(a, b, c)))
	is.Equal([]int{1, 4}, sets.Sorted(sets.SymmetricDifference(a, b)))
	is.Equal([]int{1, 2, 3}, sets.Sorted(a), "inputs are not modified")

	// Regression: Intersection with one argument returned the argument
	// itself, so changing the result changed the input.
	got := sets.Intersection(a)
	got.Add(99)
	is.False(a.Has(99))

	is.True(sets.Equal(a, sets.Of(3, 2, 1)))
	is.False(sets.Equal(a, b))
	is.True(sets.IsSubset(sets.Of(1, 2), a))
	is.True(sets.IsSubset(a, a))
	is.False(sets.IsProperSubset(a, a))
	is.True(sets.IsProperSubset(sets.Of(1), a))
	is.True(sets.IsSuperset(a, sets.Of(1)))
	is.True(sets.IsProperSuperset(a, sets.Of(1)))
	is.True(sets.IsDisjoint(a, sets.Of(9)))
	is.False(sets.IsDisjoint(a, c))

	var empty sets.Set[int]
	is.Equal(0, sets.Union(&empty).Len())
	is.True(sets.IsSubset(&empty, a))
}

func TestSync(t *testing.T) {
	is := assert.New(t)
	src := sets.Of(1)
	s := sets.NewSync(src)
	src.Add(2)
	is.False(s.Has(2), "NewSync copies its argument")

	// Regression: NewSync(nil) panicked on first use.
	var wg sync.WaitGroup
	n := sets.NewSync[int](nil)
	for i := range 100 {
		wg.Go(func() {
			n.Add(i)
			n.Has(i)
			n.Len()
		})
	}
	wg.Wait()
	is.Equal(100, n.Len())
	is.Equal(100, n.Clone().Len())

	var zero sets.Sync[string]
	is.True(zero.Add("a"))
	is.Equal("{a}", zero.String())
}

func ExampleUnique() {
	fmt.Println(sets.Unique([]string{"b", "a", "b", "c", "a"}))
	// Output: [b a c]
}

func ExampleIntersection() {
	admins := sets.Of("alice", "bob")
	active := sets.Of("bob", "carol")
	fmt.Println(sets.Sorted(sets.Intersection(admins, active)))
	// Output: [bob]
}
