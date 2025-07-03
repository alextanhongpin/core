package sliceutil

import "golang.org/x/exp/constraints"

// Sum calculates the sum of all elements in the slice.
func Sum[T constraints.Integer | constraints.Float](numbers []T) T {
	var total T
	for _, n := range numbers {
		total += n
	}
	return total
}

// Product calculates the product of all elements in the slice.
// Returns 1 for empty slices.
func Product[T constraints.Integer | constraints.Float](numbers []T) T {
	if len(numbers) == 0 {
		return 1
	}

	var result T = 1
	for _, n := range numbers {
		result *= n
	}
	return result
}

// Min finds the minimum value in the slice.
// Returns zero value and false for empty slices.
func Min[T constraints.Ordered](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}

	min := slice[0]
	for _, v := range slice[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// Max finds the maximum value in the slice.
// Returns zero value and false for empty slices.
func Max[T constraints.Ordered](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}

	max := slice[0]
	for _, v := range slice[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// Average calculates the arithmetic mean of the slice.
// Returns zero and false for empty slices.
func Average[T constraints.Integer | constraints.Float](numbers []T) (float64, bool) {
	if len(numbers) == 0 {
		return 0, false
	}

	sum := Sum(numbers)
	return float64(sum) / float64(len(numbers)), true
}
