// Package sets provides a generic Set implementation for any ordered comparable type.
// Sets are collections of unique elements that support common set operations like
// union, intersection, difference, and subset testing.
//
// This implementation is optimized for performance and provides a clean API
// similar to mathematical set operations.
package sets

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/constraints"
)

// OrderedComparable represents types that are both ordered and comparable.
// This constraint allows sets to be sorted and compared efficiently.
type OrderedComparable interface {
	constraints.Ordered
	comparable
}

// Set represents a collection of unique elements of type T.
// The zero value of Set is an empty set ready to use.
type Set[T OrderedComparable] struct {
	values map[T]struct{}
}

// New creates a new set containing the given elements.
// Duplicate elements are automatically removed.
//
// Example:
//
//	s := sets.New(1, 2, 3, 2, 1) // Set contains {1, 2, 3}
func New[T OrderedComparable](ts ...T) *Set[T] {
	values := make(map[T]struct{}, len(ts))
	for _, t := range ts {
		values[t] = struct{}{}
	}

	return &Set[T]{
		values: values,
	}
}

// FromSlice creates a new set from a slice, removing duplicates.
//
// Example:
//
//	slice := []int{1, 2, 3, 2, 1}
//	s := sets.FromSlice(slice) // Set contains {1, 2, 3}
func FromSlice[T OrderedComparable](slice []T) *Set[T] {
	return New(slice...)
}

// Add adds one or more elements to the set.
// Adding existing elements has no effect.
//
// Example:
//
//	s.Add(4, 5, 6)
func (s *Set[T]) Add(ts ...T) {
	if s.values == nil {
		s.values = make(map[T]struct{})
	}
	for _, t := range ts {
		s.values[t] = struct{}{}
	}
}

// Delete removes one or more elements from the set.
// Removing non-existent elements has no effect.
//
// Example:
//
//	s.Delete(1, 2)
func (s *Set[T]) Delete(ts ...T) {
	if s.values == nil {
		return
	}
	for _, t := range ts {
		delete(s.values, t)
	}
}

// Clear removes all elements from the set.
//
// Example:
//
//	s.Clear() // Set becomes empty
func (s *Set[T]) Clear() {
	if s.values == nil {
		return
	}
	for k := range s.values {
		delete(s.values, k)
	}
}

// Len returns the number of elements in the set.
//
// Example:
//
//	count := s.Len()
func (s *Set[T]) Len() int {
	if s.values == nil {
		return 0
	}
	return len(s.values)
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
func (s *Set[T]) Has(t T) bool {
	if s.values == nil {
		return false
	}
	_, ok := s.values[t]
	return ok
}

// Contains is an alias for Has for better readability.
//
// Example:
//
//	if s.Contains(42) { /* element exists */ }
func (s *Set[T]) Contains(t T) bool {
	return s.Has(t)
}

// All returns all elements in the set as a sorted slice.
// The order is guaranteed to be consistent across calls.
//
// Example:
//
//	elements := s.All() // []int{1, 2, 3, 4}
func (s *Set[T]) All() []T {
	if s.values == nil {
		return []T{}
	}

	res := make([]T, 0, len(s.values))
	for t := range s.values {
		res = append(res, t)
	}

	slices.Sort(res)
	return res
}

// ToSlice is an alias for All for better API consistency.
//
// Example:
//
//	slice := s.ToSlice()
func (s *Set[T]) ToSlice() []T {
	return s.All()
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
	if s.values == nil {
		return New[T]()
	}

	newSet := New[T]()
	for t := range s.values {
		newSet.Add(t)
	}
	return newSet
}

// Intersect returns a new set containing elements that exist in both sets.
// The operation is commutative: A.Intersect(B) == B.Intersect(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := a.Intersect(b) // {2, 3}
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	if s.values == nil || other.values == nil {
		return New[T]()
	}

	// Optimize by iterating over the smaller set
	if s.Len() > other.Len() {
		return other.Intersect(s)
	}

	result := New[T]()
	for t := range s.values {
		if other.Has(t) {
			result.Add(t)
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
//	c := a.Union(b) // {1, 2, 3, 4}
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	if s.values == nil {
		if other.values == nil {
			return New[T]()
		}
		return other.Clone()
	}
	if other.values == nil {
		return s.Clone()
	}

	result := s.Clone()
	for t := range other.values {
		result.Add(t)
	}

	return result
}

// Difference returns a new set containing elements in this set but not in the other.
// The operation is not commutative: A.Difference(B) != B.Difference(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := a.Difference(b) // {1}
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	if s.values == nil {
		return New[T]()
	}
	if other.values == nil {
		return s.Clone()
	}

	result := New[T]()
	for t := range s.values {
		if !other.Has(t) {
			result.Add(t)
		}
	}

	return result
}

// SymmetricDifference returns a new set containing elements in either set but not in both.
// The operation is commutative: A.SymmetricDifference(B) == B.SymmetricDifference(A).
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(2, 3, 4)
//	c := a.SymmetricDifference(b) // {1, 4}
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	return s.Difference(other).Union(other.Difference(s))
}

// Equal returns true if both sets contain exactly the same elements.
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(3, 2, 1)
//	equal := a.Equal(b) // true
func (s *Set[T]) Equal(other *Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}

	if s.values == nil && other.values == nil {
		return true
	}
	if s.values == nil || other.values == nil {
		return false
	}

	for t := range s.values {
		if !other.Has(t) {
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
//	isSubset := a.IsSubset(b) // true
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	if s.values == nil {
		return true // empty set is subset of any set
	}
	if other.values == nil {
		return s.IsEmpty()
	}

	for t := range s.values {
		if !other.Has(t) {
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
//	isSuperset := a.IsSuperset(b) // true
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// IsProperSubset returns true if this set is a subset of the other but not equal to it.
//
// Example:
//
//	a := sets.New(1, 2)
//	b := sets.New(1, 2, 3)
//	isProperSubset := a.IsProperSubset(b) // true
func (s *Set[T]) IsProperSubset(other *Set[T]) bool {
	return s.IsSubset(other) && !s.Equal(other)
}

// IsProperSuperset returns true if this set is a superset of the other but not equal to it.
//
// Example:
//
//	a := sets.New(1, 2, 3)
//	b := sets.New(1, 2)
//	isProperSuperset := a.IsProperSuperset(b) // true
func (s *Set[T]) IsProperSuperset(other *Set[T]) bool {
	return s.IsSuperset(other) && !s.Equal(other)
}

// IsDisjoint returns true if the sets have no elements in common.
//
// Example:
//
//	a := sets.New(1, 2)
//	b := sets.New(3, 4)
//	isDisjoint := a.IsDisjoint(b) // true
func (s *Set[T]) IsDisjoint(other *Set[T]) bool {
	if s.values == nil || other.values == nil {
		return true
	}

	// Check the smaller set for efficiency
	if s.Len() > other.Len() {
		return other.IsDisjoint(s)
	}

	for t := range s.values {
		if other.Has(t) {
			return false
		}
	}
	return true
}

// ForEach iterates over all elements in the set, calling the provided function for each.
// The iteration order is not guaranteed to be consistent.
//
// Example:
//
//	s.ForEach(func(element int) {
//	    fmt.Println(element)
//	})
func (s *Set[T]) ForEach(fn func(T)) {
	if s.values == nil {
		return
	}
	for t := range s.values {
		fn(t)
	}
}

// Filter returns a new set containing only elements that satisfy the predicate.
//
// Example:
//
//	s := sets.New(1, 2, 3, 4, 5)
//	evens := s.Filter(func(x int) bool { return x%2 == 0 }) // {2, 4}
func (s *Set[T]) Filter(predicate func(T) bool) *Set[T] {
	result := New[T]()
	if s.values == nil {
		return result
	}

	for t := range s.values {
		if predicate(t) {
			result.Add(t)
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
	if s.values == nil {
		return false
	}

	for t := range s.values {
		if predicate(t) {
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
	if s.values == nil {
		return true // vacuously true for empty set
	}

	for t := range s.values {
		if !predicate(t) {
			return false
		}
	}
	return true
}
