package throttle

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type throttler interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

func Func[K, V any](fn fun[K, V], t throttler) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		err = t.Do(ctx, func(ctx context.Context) error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
