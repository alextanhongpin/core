// sink collects the result.
package pipeline

import (
	"context"
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

type BatchStopper interface {
	Stop()
}

type BatchFlusher interface {
	Flush()
}

type BatchStopFlusher interface {
	BatchStopper
	BatchFlusher
}

type batcher[T comparable] struct {
	stop   func()
	flush  func()
	limit  int
	period time.Duration
}

func newBatcher[T comparable](limit int, period time.Duration, in chan T, fn func([]T)) *batcher[T] {
	b := &batcher[T]{
		limit:  limit,
		period: period,
	}
	b.init(in, fn)
	return b
}

func (b *batcher[T]) init(in <-chan T, fn func([]T)) {
	var (
		limit  = b.limit
		period = b.period
	)

	fl := make(chan struct{})

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

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer flush()

		t := time.NewTicker(period)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case k, ok := <-in:
				if !ok {
					return
				}

				_, ok = cache[k]
				if ok {
					continue
				}
				cache[k] = struct{}{}
				t.Reset(period)

				if len(cache) >= limit {
					flush()
				}
			case <-fl:
				t.Reset(period)
				flush()
			case <-t.C:
				flush()
			}
		}
	}()

	b.stop = func() {
		cancel()
		wg.Wait()
	}

	b.flush = func() {
		fl <- struct{}{}
	}
}

func (b *batcher[T]) Stop() {
	b.stop()
}

func (b *batcher[T]) Flush() {
	b.flush()
}

func Batch[T comparable](n int, period time.Duration, in chan T, fn func([]T)) BatchStopFlusher {
	return newBatcher[T](n, period, in, fn)
}
