// package op contains useful control flow operations.
package op

func If[T any](cond bool, a, b T) T {
	if cond {
		return a
	}

	return b
}

// IfZero returns a value if a is non-zero, otherwise it returns b.
// It is possible for both a and b to be zero.
func IfZero[T comparable](a, b T) T {
	var t T
	return If(a == t, b, a)
}
