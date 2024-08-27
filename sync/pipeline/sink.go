// sink collects the result.
package pipeline

import (
	"sync"
	"time"
)

func Reduce[T, V any](in chan T, fn func(T, V) (V, error), start V) (v V, err error) {
	v = start

	for i := range in {
		v, err = fn(i, v)
		if err != nil {
			return
		}
	}

	return
}

func Collect[T any](in chan T) []T {
	var res []T
	for v := range in {
		res = append(res, v)
	}

	return res
}

func Batch[T comparable](limit int, period time.Duration, in <-chan T, fn func([]T)) func() {
	cache := make(map[T]struct{})
	flush := func() {
		keys := make([]T, 0, len(cache))
		for k := range cache {
			keys = append(keys, k)
		}
		clear(cache)

		if len(keys) == 0 {
			return
		}

		fn(keys)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer flush()

		t := time.NewTicker(period)
		defer t.Stop()

		for {
			select {
			case k, ok := <-in:
				if !ok {
					return
				}

				if _, ok := cache[k]; ok {
					continue
				}
				cache[k] = struct{}{}

				if len(cache) >= limit {
					t.Reset(period)
					flush()
				}
			case <-t.C:
				flush()
			}
		}
	}()

	return func() {

		wg.Wait()
	}
}
