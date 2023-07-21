package futures

import (
	"context"
	"runtime"

	"golang.org/x/sync/semaphore"
)

var maxWorkers = runtime.GOMAXPROCS(0)

func Join[T any](ctx context.Context, futures ...Func[T]) []result[T] {
	sem := semaphore.NewWeighted(int64(maxWorkers))

	res := make([]result[T], len(futures))
	for i := 0; i < len(futures); i++ {
		i := i // https://golang.org/doc/faq#closures_and_goroutines

		fn := futures[i]
		if err := sem.Acquire(ctx, 1); err != nil {
			res[i] = fn.Await(ctx)
			continue
		}

		go func() {
			defer sem.Release(1)

			v, err := fn(ctx)
			res[i] = result[T]{
				data: v,
				err:  err,
			}
		}()
	}

	return res
}

type future[T any] interface {
	Await(ctx context.Context) result[T]
}

type Func[T any] func(ctx context.Context) (T, error)

func (f Func[T]) Await(ctx context.Context) result[T] {
	ch := make(chan result[T])

	go func() {
		v, err := f(ctx)
		ch <- result[T]{
			data: v,
			err:  err,
		}
	}()

	return <-ch
}

type result[T any] struct {
	data T
	err  error
}

func (r *result[T]) Unwrap() (T, error) {
	return r.data, r.err
}
