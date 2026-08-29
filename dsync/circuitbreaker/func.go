package circuitbreaker

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type circuitbreaker interface {
	Do(ctx context.Context, key string, fn func() error) error
}

func Func[K, V any](fn fun[K, V], cb circuitbreaker, keyFn func(context.Context, K) (string, error)) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		key, err := keyFn(ctx, req)
		if err != nil {
			var zero V
			return zero, err
		}
		err = cb.Do(ctx, key, func() error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
