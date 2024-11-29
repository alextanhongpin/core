package number

import "golang.org/x/exp/constraints"

type Number interface {
	constraints.Integer | constraints.Float
}

func Clip[T Number](lo, hi, v T) T {
	return min(hi, max(lo, v))
}
