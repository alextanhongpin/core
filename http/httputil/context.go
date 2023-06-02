package httputil

import (
	"context"
)

type Context[T any] string

func (k Context[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

func (k Context[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}
