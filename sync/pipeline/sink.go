// sink collects the result.
package pipeline

// Reduce folds values until closure or callback error. On error, the caller
// must cancel the source and drain in (for example with Flush), or otherwise
// arrange shutdown; upstream stages can remain blocked on sends.
func Reduce[T, V any](in <-chan T, fn func(T, V) (V, error), start V) (v V, err error) {
	v = start

	for i := range in {
		v, err = fn(i, v)
		if err != nil {
			return
		}
	}

	return
}

func Collect[T any](in <-chan T) []T {
	var res []T
	for v := range in {
		res = append(res, v)
	}

	return res
}

func Sink[T any](in <-chan T, fn func(T)) {
	for v := range in {
		fn(v)
	}
}

func Flush[T any](in <-chan T) {
	for range in {
	}
}

func Count[T any](in <-chan T) int {
	var i int
	for range in {
		i++
	}

	return i
}
