package httputil

import (
	"context"
	"errors"
	"fmt"
)

var ErrContextKeyNotFound = errors.New("httputil: context key not found")

type Context[T any] string

func (k Context[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

func (k Context[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}

func (k Context[T]) MustValue(ctx context.Context) T {
	t, ok := k.Value(ctx)
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrContextKeyNotFound, k))
	}

	return t
}
