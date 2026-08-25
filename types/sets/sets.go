// Package sets provides a generic Set implementation for any ordered comparable type.
// Sets are collections of unique elements that support common set operations like
// union, intersection, difference, and subset testing.
//
// This implementation is optimized for performance and provides a clean API
// similar to mathematical set operations.
package sets

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
)

func Unique[T cmp.Ordered](vals []T) []T {
	return From(vals).All()
}

// Set represents a collection of unique elements of type T.
// The zero value of Set is an empty set ready to use.
type Set[T cmp.Ordered] struct {
	vals map[T]struct{}
}

// New creates a new set containing the given elements.
// Duplicate elements are automatically removed.
//
// Example:
//
//	s := sets.New(1, 2, 3, 2, 1) // Set contains {1, 2, 3}
func New[T cmp.Ordered]() *Set[T] {
	return &Set[T]{
		vals: make(map[T]struct{}),
	}
}

// From creates a new set from a slice, removing duplicates.
//
// Example:
//
//	slice := []int{1, 2, 3, 2, 1}
//	s := sets.From(slice) // Set contains {1, 2, 3}
func From[T cmp.Ordered](slice []T) *Set[T] {
	set := New[T]()
	set.AddMany(slice...)
	return set
}

// Of is an alias for New, providing a more readable way to create sets.
//
// Example:
//
//	s := sets.Of(1, 2, 3) // Set contains {1, 2, 3}
func Of[T cmp.Ordered](ts ...T) *Set[T] {
	// Alias for New for better readability
	return From(ts)
}

// Add adds one or more elements to the set.
// Adding existing elements has no effect.
//
// Example:
//
//	s.Add(4, 5, 6)
func (s *Set[T]) Add(v T) bool {
	if s.vals == nil {
		s.vals = make(map[T]struct{})
	}

	if _, ok := s.vals[v]; !ok {
		s.vals[v] = struct{}{}
		return true
	}

	return false
}

func (s *Set[T]) AddMany(vs ...T) int {
	var count int
	for _, v := range vs {
		if s.Add(v) {
			count++
		}
	}
	return count
}

// Remove removes one or more elements from the set.
// Removing non-existent elements has no effect.
//
// Example:
//
//	s.Remove(1, 2)
func (s *Set[T]) Remove(v T) bool {
	if _, ok := s.vals[v]; ok {
		delete(s.vals, v)
		return true
	}

	return false
}

func (s *Set[T]) RemoveMany(vs ...T) int {
	var count int
	for _, v := range vs {
		if s.Remove(v) {
			count++
		}
	}
	return count
}

// Clear removes all elements from the set.
//
// Example:
//
//	s.Clear() // Set becomes empty
func (s *Set[T]) Clear() {
	clear(s.vals)
}

// Len returns the number of elements in the set.
//
// Example:
//
//	count := s.Len()
func (s *Set[T]) Len() int {
	if s.vals == nil {
		return 0
	}
	return len(s.vals)
}

// IsEmpty returns true if the set contains no elements.
//
// Example:
//
//	if s.IsEmpty() { /* handle empty set */ }
func (s *Set[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Has returns true if the set contains the given element.
//
// Example:
//
//	if s.Has(42) { /* element exists */ }
func (s *Set[T]) Has(v T) bool {
	if s.vals == nil {
		return false
	}
	_, ok := s.vals[v]
	return ok
}

// All returns the sets in consistent order.
//
// Example:
//
//	slice := s.All()
func (s *Set[T]) All() []T {
	var res []T
	for k := range s.vals {
		res = append(res, k)
	}
	return res
}

// String returns a string representation of the set.
//
// Example:
//
//	fmt.Printf("Set: %s\n", s) // Set: {1, 2, 3}
func (s *Set[T]) String() string {
	if s.IsEmpty() {
		return "{}"
	}

	elements := s.All()
	strElements := make([]string, len(elements))
	for i, elem := range elements {
		strElements[i] = fmt.Sprintf("%v", elem)
	}

	return "{" + strings.Join(strElements, ", ") + "}"
}

// Clone creates a deep copy of the set.
//
// Example:
//
//	copy := s.Clone()
func (s *Set[T]) Clone() *Set[T] {
	newSet := New[T]()
	maps.Copy(newSet.vals, s.vals)
	return newSet
}

// Intersection returns a new set containing elements that exist in both sets.
// The operation is commutative: A.Intersect(B) == B.Intersect(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := sets.Intersection(a, b) // {2, 3}
func Intersection[T cmp.Ordered](a *Set[T], bs ...*Set[T]) *Set[T] {
	for _, b := range bs {
		a = intersection(a, b)
	}
	return a
}

func intersection[T cmp.Ordered](a, b *Set[T]) *Set[T] {
	// Optimize by iterating over the smaller set
	if a.Len() > b.Len() {
		return intersection(b, a)
	}

	result := New[T]()
	for k := range a.vals {
		if b.Has(k) {
			result.Add(k)
		}
	}

	return result
}

// Union returns a new set containing all elements from both sets.
// The operation is commutative: A.Union(B) == B.Union(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := sets.Union(a, b) // {1, 2, 3, 4}
func Union[T cmp.Ordered](a *Set[T], bs ...*Set[T]) *Set[T] {
	res := New[T]()
	maps.Copy(res.vals, a.vals)
	for _, b := range bs {
		maps.Copy(res.vals, b.vals)
	}
	return res
}

// Difference returns a new set containing elements in this set but not in the other.
// The operation is not commutative: A.Difference(B) != B.Difference(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := sets.Difference(a, b) // {1}
func Difference[T cmp.Ordered](a *Set[T], bs ...*Set[T]) *Set[T] {
	res := a.Clone()
	for _, b := range bs {
		for v := range b.vals {
			res.Remove(v)
		}
	}
	return res
}

// SymmetricDifference returns a new set containing elements in either set but not in both.
// The operation is commutative: A.SymmetricDifference(B) == B.SymmetricDifference(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := sets.SymmetricDifference(a, b) // {1, 4}
func SymmetricDifference[T cmp.Ordered](a, b *Set[T]) *Set[T] {
	return Union(Difference(a, b), Difference(b, a))
}

// Equal returns true if both sets contain exactly the same elements.
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(3, 2, 1)
//	equal := sets.Equal(a, b) // true
func Equal[T cmp.Ordered](a, b *Set[T]) bool {
	if a.Len() != b.Len() {
		return false
	}

	for v := range a.vals {
		if !b.Has(v) {
			return false
		}
	}

	return true
}

// IsSubset returns true if all elements of this set are contained in the other set.
//
// Example:
//
//	a := sets.New(1, 2)
//	b := sets.New(1, 2, 3)
//	isSubset := sets.IsSubset(a, b) // true
func IsSubset[T cmp.Ordered](a, b *Set[T]) bool {
	for v := range a.vals {
		if !b.Has(v) {
			return false
		}
	}
	return true
}

// IsSuperset returns true if this set contains all elements of the other set.
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(1, 2)
//	isSuperset := sets.IsSuperset(a, b) // true
func IsSuperset[T cmp.Ordered](a, b *Set[T]) bool {
	return IsSubset(b, a)
}

// IsProperSubset returns true if this set is a subset of the other but not equal to it.
//
// Example:
//
//	a := sets.New(1, 2)
//	b := sets.New(1, 2, 3)
//	isProperSubset := sets.IsProperSubset(a, b) // true
func IsProperSubset[T cmp.Ordered](a, b *Set[T]) bool {
	return IsSubset(a, b) && !Equal(a, b)
}

// IsProperSuperset returns true if this set is a superset of the other but not equal to it.
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(1, 2)
//	isProperSuperset := sets.IsProperSuperset(a, b) // true
func IsProperSuperset[T cmp.Ordered](a, b *Set[T]) bool {
	return IsSuperset(a, b) && !Equal(a, b)
}

// IsDisjoint returns true if the sets have no elements in common.
//
// Example:
//
//	a := sets.New(1, 2)
//	b := sets.New(3, 4)
//	isDisjoint := sets.IsDisjoint(a, b) // true
func IsDisjoint[T cmp.Ordered](a, b *Set[T]) bool {
	// Check the smaller set for efficiency
	if a.Len() > b.Len() {
		return IsDisjoint(b, a)
	}

	return Intersection(a, b).Len() == 0
}

// Range iterates over all elements in the set, calling the provided function for each.
// The iteration order is not guaranteed to be consistent.
func (s *Set[T]) Range(predicate func(T)) *Set[T] {
	result := New[T]()
	if s.vals == nil {
		return result
	}

	for v := range s.vals {
		predicate(v)
	}
	return result
}

// Filter returns a new set containing only elements that satisfy the predicate.
//
// Example:
//
//	s := sets.New(1, 2, 3, 4, 5)
//	evens := s.Filter(func(x int) bool { return x%2 == 0 }) // {2, 4}
func (s *Set[T]) Filter(predicate func(T) bool) *Set[T] {
	result := New[T]()
	if s.vals == nil {
		return result
	}

	for v := range s.vals {
		if predicate(v) {
			result.Add(v)
		}
	}
	return result
}

// Any returns true if any element in the set satisfies the predicate.
//
// Example:
//
//	s := sets.New(1, 2, 3)
//	hasEven := s.Any(func(x int) bool { return x%2 == 0 }) // true
func (s *Set[T]) Any(predicate func(T) bool) bool {
	if s.vals == nil {
		return false
	}

	for v := range s.vals {
		if predicate(v) {
			return true
		}
	}
	return false
}

// Every returns true if all elements in the set satisfy the predicate.
//
// Example:
//
//	s := sets.New(2, 4, 6)
//	allEven := s.Every(func(x int) bool { return x%2 == 0 }) // true
func (s *Set[T]) Every(predicate func(T) bool) bool {
	if s.vals == nil {
		return true // vacuously true for empty set
	}

	for v := range s.vals {
		if !predicate(v) {
			return false
		}
	}
	return true
}

func Sorted[T cmp.Ordered](s *Set[T]) *SortedSet[T] {
	return &SortedSet[T]{
		Set: s,
	}
}

type SortedSet[T cmp.Ordered] struct {
	*Set[T]
}

func (s *SortedSet[T]) All() []T {
	var res []T
	for k := range s.vals {
		res = append(res, k)
	}
	slices.Sort(res)
	return res
}

// String returns a string representation of the set.
//
// Example:
//
//	fmt.Printf("Set: %s\n", s) // Set: {1, 2, 3}
func (s *SortedSet[T]) String() string {
	if s.IsEmpty() {
		return "{}"
	}

	elements := s.All()
	strElements := make([]string, len(elements))
	for i, elem := range elements {
		strElements[i] = fmt.Sprintf("%v", elem)
	}

	return "{" + strings.Join(strElements, ", ") + "}"
}
