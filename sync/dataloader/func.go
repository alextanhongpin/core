package dataloader

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type dataloader[K comparable, V any] interface {
	Load(K) (V, error)
}

func Func[K comparable, V, T any](fn fun[V, T], dl dataloader[K, V]) fun[K, T] {
	return func(ctx context.Context, k K) (T, error) {
		v, err := dl.Load(k)
		if err != nil {
			var zero T
			return zero, err
		}

		return fn(ctx, v)
	}
}
