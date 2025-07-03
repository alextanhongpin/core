package sliceutil

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
func AllIndex[T any](slice []T, predicate func(int, T) bool) bool {
	if len(slice) == 0 {
		return false
	}

	for i, item := range slice {
		if !predicate(i, item) {
			return false
		}
	}

	return true
}

// Any returns true if any element satisfies the predicate.
// Returns false for empty slices.
func Any[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}

	return false
}

// AnyIndex returns true if any element satisfies the index-based predicate.
// Returns false for empty slices.
func AnyIndex[T any](slice []T, predicate func(int, T) bool) bool {
	for i, item := range slice {
		if predicate(i, item) {
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
func NoneIndex[T any](slice []T, predicate func(int, T) bool) bool {
	for i, item := range slice {
		if predicate(i, item) {
			return false
		}
	}

	return true
}
