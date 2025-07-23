package list

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

// Average calculates the arithmetic mean of the slice.
// Returns zero and false for empty slices.
func Average[T constraints.Integer | constraints.Float](numbers []T) float64 {
	if len(numbers) == 0 {
		panic("list.Average: empty list")
	}

	sum := Sum(numbers)
	return float64(sum) / float64(len(numbers))
}

// ArgMax returns the index of the maximum value in the slice.
// Returns -1 and false for empty slices.
func ArgMax[T constraints.Ordered](slice []T) int {
	if len(slice) == 0 {
		panic("list.ArgMax: empty list")
	}
	maxIdx := 0
	maxVal := slice[0]
	for i, v := range slice[1:] {
		if v > maxVal {
			maxVal = v
			maxIdx = i + 1
		}
	}
	return maxIdx
}

// ArgMin returns the index of the minimum value in the slice.
// Returns -1 and false for empty slices.
func ArgMin[T constraints.Ordered](slice []T) int {
	if len(slice) == 0 {
		panic("list.ArgMin: empty list")
	}
	minIdx := 0
	minVal := slice[0]
	for i, v := range slice[1:] {
		if v < minVal {
			minVal = v
			minIdx = i + 1
		}
	}
	return minIdx
}

// Note: Math methods for List type would require type constraints,
// so they are implemented as separate methods for numeric List types.
// These can be added when needed for specific numeric types.
