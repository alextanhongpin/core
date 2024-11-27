package contextkey_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/http/contextkey"
	"github.com/stretchr/testify/assert"
)

func TestContextKey(t *testing.T) {
	u := contextkey.Key[int]("user")

	ctx := u.WithValue(context.Background(), 42)
	n, ok := u.Value(ctx)
	is := assert.New(t)
	is.True(ok)
	is.Equal(42, n)

	v := contextkey.Key[int]("user")
	n, ok = v.Value(ctx)
	is.True(ok)
	is.Equal(42, n)
}
