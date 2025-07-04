// Package sliceutil provides utilities for slice operations that are not
// available in the standard library's slices package.
package sliceutil

import "slices"

// Map transforms each element of the slice using the provided function.
func Map[T, U any](slice []T, transform func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

// MapIndex transforms each element of the slice using the provided function with index.
func MapIndex[T, U any](slice []T, transform func(int, T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = transform(i, item)
	}
	return result
}

// MapError transforms each element of the slice using the provided function that can return an error.
// Returns the first error encountered, or nil if all transformations succeed.
func MapError[T, U any](slice []T, transform func(T) (U, error)) ([]U, error) {
	result := make([]U, len(slice))
	for i, item := range slice {
		transformed, err := transform(item)
		if err != nil {
			return nil, err
		}
		result[i] = transformed
	}
	return result, nil
}

// MapIndexError transforms each element of the slice using the provided function with index that can return an error.
// Returns the first error encountered, or nil if all transformations succeed.
func MapIndexError[T, U any](slice []T, transform func(int, T) (U, error)) ([]U, error) {
	result := make([]U, len(slice))
	for i, item := range slice {
		transformed, err := transform(i, item)
		if err != nil {
			return nil, err
		}
		result[i] = transformed
	}
	return result, nil
}

// FlatMap applies the transform function to each element and flattens the results.
func FlatMap[T, U any](slice []T, transform func(T) []U) []U {
	var result []U
	for _, item := range slice {
		result = append(result, transform(item)...)
	}
	return result
}

// Reduce applies a function against all elements in the slice to reduce it to a single value.
func Reduce[T, U any](slice []T, initial U, reducer func(U, T) U) U {
	result := initial
	for _, item := range slice {
		result = reducer(result, item)
	}
	return result
}

// ReduceIndex applies a function against all elements in the slice with index to reduce it to a single value.
func ReduceIndex[T, U any](slice []T, initial U, reducer func(U, int, T) U) U {
	result := initial
	for i, item := range slice {
		result = reducer(result, i, item)
	}
	return result
}

// Dedup removes duplicate elements from the slice, preserving order of first occurrence.
func Dedup[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return slice
	}

	seen := make(map[T]bool, len(slice))
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// DedupFunc removes duplicate elements from the slice based on a key function.
// Preserves order of first occurrence.
func DedupFunc[T any, K comparable](slice []T, keyFunc func(T) K) []T {
	if len(slice) == 0 {
		return slice
	}

	seen := make(map[K]bool, len(slice))
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		key := keyFunc(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// Reverse returns a new slice with elements in reverse order.
func Reverse[T any](slice []T) []T {
	if len(slice) <= 1 {
		return slices.Clone(slice)
	}

	result := make([]T, len(slice))
	for i, item := range slice {
		result[len(slice)-1-i] = item
	}
	return result
}

// Chunk splits the slice into chunks of the specified size.
// The last chunk may be smaller if the slice length is not divisible by the chunk size.
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}

	if len(slice) == 0 {
		return [][]T{}
	}

	chunks := make([][]T, 0, (len(slice)+size-1)/size)
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// Flatten flattens a slice of slices into a single slice.
func Flatten[T any](slices [][]T) []T {
	var result []T
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

// GroupBy groups elements of the slice by the result of the key function.
func GroupBy[T any, K comparable](slice []T, keyFunc func(T) K) map[K][]T {
	groups := make(map[K][]T)
	for _, item := range slice {
		key := keyFunc(item)
		groups[key] = append(groups[key], item)
	}
	return groups
}

// Partition splits the slice into two slices based on a predicate.
// The first slice contains elements that satisfy the predicate,
// the second contains elements that don't.
func Partition[T any](slice []T, predicate func(T) bool) ([]T, []T) {
	var trueSlice, falseSlice []T

	for _, item := range slice {
		if predicate(item) {
			trueSlice = append(trueSlice, item)
		} else {
			falseSlice = append(falseSlice, item)
		}
	}

	return trueSlice, falseSlice
}

// Zip combines elements from two slices into pairs.
// The resulting slice length is the minimum of the two input slice lengths.
func Zip[T, U any](slice1 []T, slice2 []U) []struct {
	First  T
	Second U
} {
	minLen := len(slice1)
	if len(slice2) < minLen {
		minLen = len(slice2)
	}

	result := make([]struct {
		First  T
		Second U
	}, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = struct {
			First  T
			Second U
		}{slice1[i], slice2[i]}
	}

	return result
}

// Unzip separates a slice of pairs into two separate slices.
func Unzip[T, U any](pairs []struct {
	First  T
	Second U
}) ([]T, []U) {
	first := make([]T, len(pairs))
	second := make([]U, len(pairs))

	for i, pair := range pairs {
		first[i] = pair.First
		second[i] = pair.Second
	}

	return first, second
}
