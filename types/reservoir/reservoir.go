package reservoir

import "math/rand/v2"

// ReservoirSampling selects k items from a stream of unknown size
type ReservoirSampling[T any] struct {
	k         int
	i         int
	reservoir []T
}

func NewReservoirSampling[T any](k int) *ReservoirSampling[T] {
	return &ReservoirSampling[T]{
		k:         k,
		reservoir: make([]T, k),
	}
}

func (r *ReservoirSampling[T]) Add(v T) {
	if r.i < r.k {
		r.reservoir[r.i] = v
	} else {
		j := rand.N(r.i + 1)
		if j < r.k {
			r.reservoir[j] = v
		}
	}
	r.i++
}
