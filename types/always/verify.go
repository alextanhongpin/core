package always

import "context"

type Verifier[T any] func(ctx context.Context, t T) error

func Verify[T any](ctx context.Context, t T, fns ...Verifier[T]) error {
	for i := 0; i < len(fns); i++ {
		if err := fns[i](ctx, t); err != nil {
			return err
		}
	}

	return nil
}
