// Package contextkey provides typed context keys.
//
// Each key returned by New is distinct, even when two keys share a name and
// a type, so keys defined in different packages never collide:
//
//	var UserIDKey = contextkey.New[int]("user_id")
//
//	ctx = UserIDKey.WithValue(ctx, 123)
//	userID, ok := UserIDKey.Value(ctx) // 123, true
package contextkey

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotFound is the panic value (wrapped) of MustValue when the key is
// missing.
var ErrNotFound = errors.New("contextkey: key not found")

// Key is a context key for values of type T. Create keys with New and compare
// them by pointer; the name is only used for debugging.
type Key[T any] struct {
	name string
}

// New returns a new key. Every call returns a different key.
func New[T any](name string) *Key[T] {
	return &Key[T]{name: name}
}

// String returns the key name.
func (k *Key[T]) String() string {
	return k.name
}

// WithValue returns a copy of ctx that carries t under k.
func (k *Key[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

// WithValueNX is like WithValue, but leaves ctx unchanged and returns false
// when ctx already carries a value for k.
func (k *Key[T]) WithValueNX(ctx context.Context, t T) (context.Context, bool) {
	if _, ok := k.Value(ctx); ok {
		return ctx, false
	}

	return context.WithValue(ctx, k, t), true
}

// Value returns the value stored under k and whether it was present.
func (k *Key[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}

// MustValue is like Value, but panics with an error wrapping ErrNotFound when
// the key is missing.
func (k *Key[T]) MustValue(ctx context.Context) T {
	t, ok := k.Value(ctx)
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrNotFound, k.name))
	}

	return t
}
