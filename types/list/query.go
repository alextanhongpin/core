package list

// Find returns the first element that satisfies the predicate.
// Returns zero value and false if no element is found.
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	for _, item := range slice {
		if predicate(item) {
			return item, true
		}
	}

	var zero T
	return zero, false
}

// FindIndex returns the first element and its index that satisfies the predicate.
// Returns zero value, -1, and false if no element is found.
func FindIndex[T any](slice []T, predicate func(T) bool) (T, int, bool) {
	for i, item := range slice {
		if predicate(item) {
			return item, i, true
		}
	}

	var zero T
	return zero, -1, false
}

// FindLast returns the last element that satisfies the predicate.
// Returns zero value and false if no element is found.
func FindLast[T any](slice []T, predicate func(T) bool) (T, bool) {
	for i := len(slice) - 1; i >= 0; i-- {
		if predicate(slice[i]) {
			return slice[i], true
		}
	}

	var zero T
	return zero, false
}

// Filter returns a new slice with all elements that satisfy the predicate.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// FilterIndex returns a new slice with all elements that satisfy the index-based predicate.
func FilterIndex[T any](slice []T, predicate func(int) bool) []T {
	result := make([]T, 0, len(slice))
	for i, item := range slice {
		if predicate(i) {
			result = append(result, item)
		}
	}
	return result
}

// Head returns the first element of the slice.
// Returns zero value and false for empty slices.
func Head[T any](slice []T) (T, bool) {
	if len(slice) > 0 {
		return slice[0], true
	}

	var zero T
	return zero, false
}

// Tail returns the last element of the slice.
// Returns zero value and false for empty slices.
func Tail[T any](slice []T) (T, bool) {
	if len(slice) > 0 {
		return slice[len(slice)-1], true
	}

	var zero T
	return zero, false
}

// Take returns a slice with the first n elements.
// If n is greater than the slice length, returns the entire slice.
func Take[T any](slice []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	if n >= len(slice) {
		return slice
	}
	return slice[:n]
}

// TakeLast returns a slice with the last n elements.
// If n is greater than the slice length, returns the entire slice.
func TakeLast[T any](slice []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	if n >= len(slice) {
		return slice
	}
	return slice[len(slice)-n:]
}

// Drop returns a slice with the first n elements removed.
// If n is greater than the slice length, returns an empty slice.
func Drop[T any](slice []T, n int) []T {
	if n <= 0 {
		return slice
	}
	if n >= len(slice) {
		return []T{}
	}
	return slice[n:]
}

// DropLast returns a slice with the last n elements removed.
// If n is greater than the slice length, returns an empty slice.
func DropLast[T any](slice []T, n int) []T {
	if n <= 0 {
		return slice
	}
	if n >= len(slice) {
		return []T{}
	}
	return slice[:len(slice)-n]
}

// IndexOf returns the index of the first occurrence of the element.
// Returns -1 if the element is not found.
func IndexOf[T comparable](slice []T, element T) int {
	for i, item := range slice {
		if item == element {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence of the element.
// Returns -1 if the element is not found.
func LastIndexOf[T comparable](slice []T, element T) int {
	for i := len(slice) - 1; i >= 0; i-- {
		if slice[i] == element {
			return i
		}
	}
	return -1
}

// Chainable methods for List type

// Find returns the first element that satisfies the predicate.
func (l *List[T]) Find(predicate func(T) bool) (T, bool) {
	return Find(l.data, predicate)
}

// FindIndex returns the first element and its index that satisfies the predicate.
func (l *List[T]) FindIndex(predicate func(T) bool) (T, int, bool) {
	return FindIndex(l.data, predicate)
}

// FindLast returns the last element that satisfies the predicate.
func (l *List[T]) FindLast(predicate func(T) bool) (T, bool) {
	return FindLast(l.data, predicate)
}

// Head returns the first element of the list.
func (l *List[T]) Head() (T, bool) {
	return Head(l.data)
}

// Tail returns the last element of the list.
func (l *List[T]) Tail() (T, bool) {
	return Tail(l.data)
}

// Take returns a new list with the first n elements.
func (l *List[T]) Take(n int) *List[T] {
	result := Take(l.data, n)
	return &List[T]{data: result}
}

// TakeLast returns a new list with the last n elements.
func (l *List[T]) TakeLast(n int) *List[T] {
	result := TakeLast(l.data, n)
	return &List[T]{data: result}
}

// Drop returns a new list with the first n elements removed.
func (l *List[T]) Drop(n int) *List[T] {
	result := Drop(l.data, n)
	return &List[T]{data: result}
}

// DropLast returns a new list with the last n elements removed.
func (l *List[T]) DropLast(n int) *List[T] {
	result := DropLast(l.data, n)
	return &List[T]{data: result}
}
