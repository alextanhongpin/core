package sliceutil

// Any returns true if all of the result returns true.
func All[T any](t []T, fn func(i int) bool) bool {
	for i := 0; i < len(t); i++ {
		if !fn(i) {
			return false
		}
	}

	return true
}

// Any returns true if any of the result returns true.
func Any[T any](t []T, fn func(i int) bool) bool {
	for i := 0; i < len(t); i++ {
		if fn(i) {
			return true
		}
	}

	return false
}

// Some is an alias to Any.
func Some[T any](t []T, fn func(i int) bool) bool {
	return Any(t, fn)
}

// None returns true if none of the result returns true.
func None[T any](t []T, fn func(i int) bool) bool {
	for i := 0; i < len(t); i++ {
		if fn(i) {
			return false
		}
	}

	return true
}
