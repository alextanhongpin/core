package lock

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type locker interface {
	Do(ctx context.Context, key string, fn func(ctx context.Context) error) error
}

func Func[K, V any](fn fun[K, V], l locker, keyFn func(K) string) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		err = l.Do(ctx, keyFn(req), func(ctx context.Context) error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
