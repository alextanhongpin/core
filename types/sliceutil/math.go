package sliceutil

import "golang.org/x/exp/constraints"

func Sum[T constraints.Integer](n []T) (total T) {
	for i := 0; i < len(n); i++ {
		total += n[i]
	}

	return total
}
