package types

import (
	"context"
	"errors"
	"fmt"
)

var ErrContextNotFound = errors.New("ContextKey: not found")

type ContextKey[T any] string

func (k ContextKey[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

func (k ContextKey[T]) Value(ctx context.Context) (T, error) {
	t, ok := ctx.Value(k).(T)
	if !ok {
		return t, fmt.Errorf("%w: %s", ErrContextNotFound, k)
	}

	return t, nil
}

func (k ContextKey[T]) MustValue(ctx context.Context) T {
	t, err := k.Value(ctx)
	if err != nil {
		panic(err)
	}

	return t
}
