// Package sets provides a generic set of comparable values with union,
// intersection, difference, and subset operations, plus a mutex-guarded
// variant for concurrent use.
package sets

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// Unique returns the distinct values of vals in order of first appearance.
// It returns nil when vals is nil. Use slices.Compact for sorted input.
func Unique[T comparable](vals []T) []T {
	if vals == nil {
		return nil
	}
	seen := make(map[T]struct{}, len(vals))
	out := make([]T, 0, len(vals))
	for _, v := range vals {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Set is an unordered collection of distinct values. The zero value is an
// empty set ready to use. A Set is not safe for concurrent use; see Sync.
//
// The package-level operations (Union, Intersection, ...) never modify their
// arguments and always return a new set.
type Set[T comparable] struct {
	vals map[T]struct{}
}

// New returns an empty set.
func New[T comparable]() *Set[T] {
	return &Set[T]{vals: make(map[T]struct{})}
}

// From returns a set holding the values of s.
func From[T comparable](s []T) *Set[T] {
	set := &Set[T]{vals: make(map[T]struct{}, len(s))}
	set.AddMany(s...)
	return set
}

// Of returns a set holding the given values.
func Of[T comparable](vs ...T) *Set[T] {
	return From(vs)
}

// Add inserts v and reports whether it was not already present.
func (s *Set[T]) Add(v T) bool {
	if s.vals == nil {
		s.vals = make(map[T]struct{})
	}
	if _, ok := s.vals[v]; ok {
		return false
	}
	s.vals[v] = struct{}{}
	return true
}

// AddMany inserts vs and returns how many were not already present.
func (s *Set[T]) AddMany(vs ...T) int {
	n := 0
	for _, v := range vs {
		if s.Add(v) {
			n++
		}
	}
	return n
}

// Remove deletes v and reports whether it was present.
func (s *Set[T]) Remove(v T) bool {
	if _, ok := s.vals[v]; !ok {
		return false
	}
	delete(s.vals, v)
	return true
}

// RemoveMany deletes vs and returns how many were present.
func (s *Set[T]) RemoveMany(vs ...T) int {
	n := 0
	for _, v := range vs {
		if s.Remove(v) {
			n++
		}
	}
	return n
}

// Clear removes every value.
func (s *Set[T]) Clear() { clear(s.vals) }

// Len returns the number of values.
func (s *Set[T]) Len() int { return len(s.vals) }

// IsEmpty reports whether the set has no values.
func (s *Set[T]) IsEmpty() bool { return len(s.vals) == 0 }

// Has reports whether v is in the set.
func (s *Set[T]) Has(v T) bool {
	_, ok := s.vals[v]
	return ok
}

// All returns the values as a new slice in unspecified order. Use Sorted for
// a deterministic order.
func (s *Set[T]) All() []T {
	return slices.Collect(maps.Keys(s.vals))
}

// Values returns an iterator over the values in unspecified order. The set
// must not be modified during iteration, except by deleting the current
// value.
func (s *Set[T]) Values() iter.Seq[T] {
	return maps.Keys(s.vals)
}

// Filter returns a new set holding the values for which keep returns true.
func (s *Set[T]) Filter(keep func(T) bool) *Set[T] {
	out := New[T]()
	for v := range s.vals {
		if keep(v) {
			out.vals[v] = struct{}{}
		}
	}
	return out
}

// Clone returns a copy of the set.
func (s *Set[T]) Clone() *Set[T] {
	out := &Set[T]{vals: make(map[T]struct{}, len(s.vals))}
	maps.Copy(out.vals, s.vals)
	return out
}

// String formats the set as "{a, b, c}" in unspecified order.
func (s *Set[T]) String() string {
	return format(s.All())
}

func format[T any](vs []T) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprint(v)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Sorted returns the values of s in ascending order.
func Sorted[T cmp.Ordered](s *Set[T]) []T {
	vs := s.All()
	slices.Sort(vs)
	return vs
}

// Intersection returns the values present in a and every set in bs. With no
// bs it returns a copy of a.
func Intersection[T comparable](a *Set[T], bs ...*Set[T]) *Set[T] {
	out := a.Clone()
	for _, b := range bs {
		for v := range out.vals {
			if !b.Has(v) {
				delete(out.vals, v)
			}
		}
	}
	return out
}

// Union returns the values present in a or any set in bs.
func Union[T comparable](a *Set[T], bs ...*Set[T]) *Set[T] {
	out := a.Clone()
	for _, b := range bs {
		maps.Copy(out.vals, b.vals)
	}
	return out
}

// Difference returns the values of a that are in none of bs.
func Difference[T comparable](a *Set[T], bs ...*Set[T]) *Set[T] {
	out := a.Clone()
	for _, b := range bs {
		for v := range b.vals {
			delete(out.vals, v)
		}
	}
	return out
}

// SymmetricDifference returns the values in exactly one of a and b.
func SymmetricDifference[T comparable](a, b *Set[T]) *Set[T] {
	return Union(Difference(a, b), Difference(b, a))
}

// Equal reports whether a and b hold the same values.
func Equal[T comparable](a, b *Set[T]) bool {
	return a.Len() == b.Len() && IsSubset(a, b)
}

// IsSubset reports whether every value of a is in b.
func IsSubset[T comparable](a, b *Set[T]) bool {
	if a.Len() > b.Len() {
		return false
	}
	for v := range a.vals {
		if !b.Has(v) {
			return false
		}
	}
	return true
}

// IsSuperset reports whether every value of b is in a.
func IsSuperset[T comparable](a, b *Set[T]) bool { return IsSubset(b, a) }

// IsProperSubset reports whether a is a subset of b and b has more values.
func IsProperSubset[T comparable](a, b *Set[T]) bool {
	return a.Len() < b.Len() && IsSubset(a, b)
}

// IsProperSuperset reports whether a is a superset of b and a has more values.
func IsProperSuperset[T comparable](a, b *Set[T]) bool { return IsProperSubset(b, a) }

// IsDisjoint reports whether a and b have no values in common.
func IsDisjoint[T comparable](a, b *Set[T]) bool {
	if a.Len() > b.Len() {
		a, b = b, a
	}
	for v := range a.vals {
		if b.Has(v) {
			return false
		}
	}
	return true
}
