// Package number clamps, range-checks, interpolates, and rescales numeric
// values of any integer or floating-point type.
//
// Functions that take a range accept its bounds in either order.
package number

import "golang.org/x/exp/constraints"

// Number is any integer or floating-point type.
type Number interface {
	constraints.Integer | constraints.Float
}

// Clip returns v limited to the closed range between lo and hi.
// The bounds may be given in either order. A NaN v is returned unchanged.
func Clip[T Number](lo, hi, v T) T {
	if lo > hi {
		lo, hi = hi, lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// InRange reports whether v lies in the closed range between lo and hi.
// The bounds may be given in either order. It is false for NaN.
func InRange[T Number](lo, hi, v T) bool {
	if lo > hi {
		lo, hi = hi, lo
	}
	return v >= lo && v <= hi
}

// Sign returns -1 if v is negative, 1 if it is positive, and 0 otherwise
// (including for zero and NaN).
func Sign[T Number](v T) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	default:
		return 0
	}
}

// Lerp linearly interpolates from a to b: t = 0 gives a and t = 1 gives b.
// t is not clamped, so values outside [0, 1] extrapolate.
func Lerp[T constraints.Float](a, b, t T) T {
	return a + t*(b-a)
}

// Map rescales v from the range [inMin, inMax] to [outMin, outMax]. The
// result is not clamped; wrap it in Clip if needed. Reversed ranges work
// (for example outMin > outMax inverts the mapping). If inMin == inMax the
// input range is empty and Map returns outMin.
func Map[T constraints.Float](v, inMin, inMax, outMin, outMax T) T {
	if inMax == inMin {
		return outMin
	}
	return outMin + (v-inMin)*(outMax-outMin)/(inMax-inMin)
}

// Normalize rescales v from [lo, hi] to [0, 1]. It is Map(v, lo, hi, 0, 1):
// unclamped, and 0 when lo == hi.
func Normalize[T constraints.Float](v, lo, hi T) T {
	return Map(v, lo, hi, 0, 1)
}
