package lock

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type locker interface {
	Do(ctx context.Context, key string, fn func(ctx context.Context) error) error
}

func Func[K, V any](fn fun[K, V], l locker, keyFn func(context.Context, K) (string, error)) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		var zero V
		key, err := keyFn(ctx, req)
		if err != nil {
			return zero, err
		}
		err = l.Do(ctx, key, func(ctx context.Context) error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
