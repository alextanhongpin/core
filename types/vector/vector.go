package vector

import "math"

type Vector []float64

func (v Vector) Dot(o []float64) float64 {
	if len(v) != len(o) {
		panic("vectors must have same length")
	}
	var res float64
	for i := range len(v) {
		res += v[i] * o[i]
	}
	return res
}

func (v Vector) Magnitude() float64 {
	var res float64
	for _, i := range v {
		res += i * i
	}

	return math.Sqrt(res)
}

func (v Vector) CosineSimilarity(o Vector) float64 {
	if v.Magnitude() == 0 || o.Magnitude() == 0 {
		return 0
	}

	num := v.Dot(o)
	den := v.Magnitude() * o.Magnitude()
	return num / den
}
