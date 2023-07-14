package sliceutil

import "golang.org/x/exp/constraints"

func Sum[T constraints.Integer](n []T) (total T) {
	for i := 0; i < len(n); i++ {
		total += n[i]
	}

	return total
}

func Min[T constraints.Integer](n []T) T {
	if len(n) == 0 {
		return 0
	}

	min := n[0]
	for i := 1; i < len(n); i++ {
		if n[i] < min {
			min = n[i]
		}
	}

	return min
}

func Max[T constraints.Integer](n []T) T {
	if len(n) == 0 {
		return 0
	}

	max := n[0]
	for i := 1; i < len(n); i++ {
		if n[i] > max {
			max = n[i]
		}
	}

	return max
}
