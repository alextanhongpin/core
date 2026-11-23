package vector

import "math"

// Vector is a dense float64 vector.
type Vector []float64

// Dot returns the dot product of v and o. It panics if their lengths differ.
func (v Vector) Dot(o Vector) float64 {
	if len(v) != len(o) {
		panic("vector: length mismatch")
	}
	var res float64
	for i := range v {
		res += v[i] * o[i]
	}
	return res
}

// Magnitude returns the Euclidean (L2) norm of v.
func (v Vector) Magnitude() float64 {
	return math.Sqrt(v.Dot(v))
}

// CosineSimilarity returns the cosine of the angle between v and o, in
// [-1, 1]. It returns 0 if either vector has zero magnitude, and panics if
// their lengths differ.
func (v Vector) CosineSimilarity(o Vector) float64 {
	dot := v.Dot(o)
	den := v.Magnitude() * o.Magnitude()
	if den == 0 {
		return 0
	}
	return dot / den
}
