package circuitbreaker

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type circuitbreaker interface {
	Do(fn func() error) error
}

func Func[K, V any](fn fun[K, V], cb circuitbreaker) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		err = cb.Do(func() error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
