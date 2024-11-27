package contextkey

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("contextkey: key not found")

type Key[T any] string

func (k Key[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

func (k Key[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}

func (k Key[T]) MustValue(ctx context.Context) T {
	t, ok := k.Value(ctx)
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrNotFound, k))
	}

	return t
}
