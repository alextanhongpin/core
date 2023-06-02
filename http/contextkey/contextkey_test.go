package contextkey_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/http/contextkey"
	"github.com/stretchr/testify/assert"
)

func TestContextKey(t *testing.T) {
	var userCtx = contextkey.Value[int]("hello")

	ctx := context.Background()
	ctx = userCtx.WithValue(ctx, 42)
	n, ok := userCtx.Value(ctx)
	assert := assert.New(t)
	assert.True(ok)
	assert.Equal(42, n)
}
