package sliceutil

func Find[T any](t []T, fn func(i int) bool) (T, bool) {
	for i := range len(t) {
		if fn(i) {
			return t[i], true
		}
	}

	var v T
	return v, false
}

func Filter[T any](t []T, fn func(i int) bool) []T {
	res := make([]T, 0, len(t))
	for i := range len(t) {
		if !fn(i) {
			continue
		}

		res = append(res, t[i])
	}

	return res
}

func Head[T any](t []T) (v T, ok bool) {
	if len(t) > 0 {
		return t[0], true
	}

	return
}

func Tail[T any](t []T) (v T, ok bool) {
	if len(t) > 0 {
		return t[len(t)-1], true
	}

	return
}

func Take[T any](t []T, n int) []T {
	if len(t) >= n {
		return t[:n]
	}

	return t
}
