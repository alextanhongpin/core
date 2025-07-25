package list

import "slices"

// All returns true if all elements satisfy the predicate.
// Returns false for empty slices.
func All[T any](slice []T, predicate func(T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for _, item := range slice {
		if !predicate(item) {
			return false
		}
	}

	return true
}

// AllIndex returns true if all elements satisfy the index-based predicate.
// Returns false for empty slices.
func AllIndex[T any](slice []T, predicate func(int) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for i := range slice {
		if !predicate(i) {
			return false
		}
	}

	return true
}

// Any returns true if any element satisfies the predicate.
// Returns false for empty slices.
func Any[T any](slice []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(slice, predicate)
}

// AnyIndex returns true if any element satisfies the index-based predicate.
// Returns false for empty slices.
func AnyIndex[T any](slice []T, predicate func(int) bool) bool {
	for i := range slice {
		if predicate(i) {
			return true
		}
	}

	return false
}

// Some is an alias to Any for JavaScript-like syntax.
func Some[T any](slice []T, predicate func(T) bool) bool {
	return Any(slice, predicate)
}

// None returns true if no elements satisfy the predicate.
// Returns true for empty slices.
func None[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return false
		}
	}

	return true
}

// NoneIndex returns true if no elements satisfy the index-based predicate.
// Returns true for empty slices.
func NoneIndex[T any](slice []T, predicate func(int) bool) bool {
	for i := range slice {
		if predicate(i) {
			return false
		}
	}

	return true
}

// Chainable methods for List type

// All returns true if all elements satisfy the predicate.
func (l *List[T]) All(predicate func(T) bool) bool {
	return All(l.data, predicate)
}

// Any returns true if any element satisfies the predicate.
func (l *List[T]) Any(predicate func(T) bool) bool {
	return Any(l.data, predicate)
}

// Some is an alias to Any for JavaScript-like syntax.
func (l *List[T]) Some(predicate func(T) bool) bool {
	return Some(l.data, predicate)
}

// None returns true if no elements satisfy the predicate.
func (l *List[T]) None(predicate func(T) bool) bool {
	return None(l.data, predicate)
}
