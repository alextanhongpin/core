// sink collects the result.
package pipeline

import "sync"

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

func Process[T any](in <-chan T, fn func(T)) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for v := range in {
			fn(v)
		}
	}()

	return wg.Wait
}
