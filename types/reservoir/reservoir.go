package reservoir

import "math/rand/v2"

// ReservoirSampling keeps a uniform random sample of at most k items from a
// stream of unknown length (Algorithm R). It is not safe for concurrent use.
type ReservoirSampling[T any] struct {
	k         int
	n         int
	reservoir []T
}

// NewReservoirSampling returns a sampler that keeps up to k items.
// It panics if k <= 0.
func NewReservoirSampling[T any](k int) *ReservoirSampling[T] {
	if k <= 0 {
		panic("reservoir: k must be positive")
	}
	return &ReservoirSampling[T]{
		k:         k,
		reservoir: make([]T, 0, k),
	}
}

// Add offers v to the sample. After n calls, every item seen so far is in the
// sample with probability min(k, n)/n.
func (r *ReservoirSampling[T]) Add(v T) {
	r.n++
	if len(r.reservoir) < r.k {
		r.reservoir = append(r.reservoir, v)
		return
	}
	if j := rand.N(r.n); j < r.k {
		r.reservoir[j] = v
	}
}

// Sample returns a copy of the current sample: min(k, Count()) items in no
// particular order.
func (r *ReservoirSampling[T]) Sample() []T {
	return append([]T(nil), r.reservoir...)
}

// Count returns the number of items offered with Add.
func (r *ReservoirSampling[T]) Count() int {
	return r.n
}
