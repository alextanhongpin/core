// Package number provides mathematical utilities and operations for numeric types.
// It includes functions for clipping values to ranges, mathematical operations,
// and numeric utilities that work across different numeric types using generics.
package number

import "golang.org/x/exp/constraints"

// Number represents any numeric type (integers and floats)
type Number interface {
	constraints.Integer | constraints.Float
}

// Clip constrains a value to be within the specified range [lo, hi].
// If the value is less than lo, returns lo.
// If the value is greater than hi, returns hi.
// Otherwise returns the value unchanged.
func Clip[T Number](lo, hi, v T) T {
	return min(hi, max(lo, v))
}

// ClipMin constrains a value to be at least the minimum value.
func ClipMin[T Number](minVal, v T) T {
	return max(minVal, v)
}

// ClipMax constrains a value to be at most the maximum value.
func ClipMax[T Number](maxVal, v T) T {
	return min(maxVal, v)
}

// InRange checks if a value is within the specified range [lo, hi] (inclusive).
func InRange[T Number](lo, hi, v T) bool {
	return v >= lo && v <= hi
}

// Abs returns the absolute value of a number.
func Abs[T Number](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

// Sign returns the sign of a number:
// -1 if negative, 0 if zero, 1 if positive.
func Sign[T Number](v T) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

// Lerp performs linear interpolation between two values.
// t should be between 0 and 1, where 0 returns a and 1 returns b.
func Lerp[T constraints.Float](a, b, t T) T {
	return a + t*(b-a)
}

// Map maps a value from one range to another.
// Maps value from range [inMin, inMax] to range [outMin, outMax].
func Map[T constraints.Float](value, inMin, inMax, outMin, outMax T) T {
	if inMax == inMin {
		return outMin // Avoid division by zero
	}
	return outMin + (value-inMin)*(outMax-outMin)/(inMax-inMin)
}

// Normalize maps a value from range [min, max] to range [0, 1].
func Normalize[T constraints.Float](value, min, max T) T {
	if max == min {
		return 0
	}
	return (value - min) / (max - min)
}

// Denormalize maps a value from range [0, 1] to range [min, max].
func Denormalize[T constraints.Float](normalizedValue, min, max T) T {
	return min + normalizedValue*(max-min)
}

// Round rounds a float to the nearest integer.
func Round[T constraints.Float](v T) T {
	if v < 0 {
		return T(int(v - 0.5))
	}
	return T(int(v + 0.5))
}

// Floor returns the largest integer value less than or equal to the input.
func Floor[T constraints.Float](v T) T {
	return T(int(v))
}

// Ceil returns the smallest integer value greater than or equal to the input.
func Ceil[T constraints.Float](v T) T {
	iv := int(v)
	if v > T(iv) {
		return T(iv + 1)
	}
	return T(iv)
}
