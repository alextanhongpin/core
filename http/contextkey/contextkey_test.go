package contextkey_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alextanhongpin/core/http/contextkey"
	"github.com/stretchr/testify/assert"
)

func TestContextKey(t *testing.T) {
	u := contextkey.New[int]("user")

	ctx := u.WithValue(context.Background(), 42)
	n, ok := u.Value(ctx)
	is := assert.New(t)
	is.True(ok)
	is.Equal(42, n)
	is.Equal(42, u.MustValue(ctx))
	is.Equal("user", u.String())
}

func TestKeysWithSameNameDoNotCollide(t *testing.T) {
	// Regression: keys were plain strings, so two packages defining a
	// Key[int]("user") read and overwrote each other's values.
	a := contextkey.New[int]("user")
	b := contextkey.New[int]("user")

	ctx := a.WithValue(context.Background(), 42)
	_, ok := b.Value(ctx)
	assert.False(t, ok)
}

func TestWithValueNX(t *testing.T) {
	k := contextkey.New[string]("k")
	ctx, ok := k.WithValueNX(context.Background(), "a")
	assert.True(t, ok)

	ctx, ok = k.WithValueNX(ctx, "b")
	assert.False(t, ok)
	assert.Equal(t, "a", k.MustValue(ctx))
}

func TestMustValuePanics(t *testing.T) {
	defer func() {
		err, _ := recover().(error)
		assert.True(t, errors.Is(err, contextkey.ErrNotFound))
	}()
	contextkey.New[int]("missing").MustValue(context.Background())
}
