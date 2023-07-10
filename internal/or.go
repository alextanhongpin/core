package internal

func Or[T comparable](a, b T) T {
	var t T
	if a == t {
		return b
	}

	return a
}
