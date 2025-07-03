package sliceutil

import "golang.org/x/exp/constraints"

func Sum[T constraints.Integer](n []T) (total T) {
	for i := range len(n) {
		total += n[i]
	}

	return total
}
