package topk

import (
	"cmp"
	"iter"
	"slices"
)

type Tuple[T1, T2 any] struct {
	T1 T1
	T2 T2
}

func TopK[T any](seq iter.Seq2[float32, T], k int) ([]T, error) {
	if k == 0 {
		panic("k must not be zero")
	}
	res := make([]Tuple[float32, T], 0, k)
	for score, val := range seq {
		t := Tuple[float32, T]{T1: score, T2: val}
		if len(res) == k {
			minI := 0
			for i := range res {
				if res[minI].T1 > res[i].T1 {
					minI = i
				}
			}
			if res[minI].T1 < score {
				res[minI] = t
			}
		} else {
			res = append(res, t)
		}
	}

	slices.SortFunc(res, func(i, j Tuple[float32, T]) int {
		return -cmp.Compare(i.T1, j.T1)
	})

	out := make([]T, len(res))
	for i, v := range res {
		out[i] = v.T2
	}

	return out, nil
}
