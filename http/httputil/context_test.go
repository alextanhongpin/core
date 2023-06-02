package httputil_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/stretchr/testify/assert"
)

func TestContextKey(t *testing.T) {
	var userCtx = httputil.Context[int]("user_ctx")

	ctx := context.Background()
	ctx = userCtx.WithValue(ctx, 42)
	n, ok := userCtx.Value(ctx)
	assert := assert.New(t)
	assert.True(ok)
	assert.Equal(42, n)
}
