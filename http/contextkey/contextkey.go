package contextkey

import (
	"context"
)

type Value[T any] string

func (k Value[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

func (k Value[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}
