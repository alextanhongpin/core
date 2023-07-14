package okay_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/types/okay"
	"github.com/stretchr/testify/assert"
)

func TestZero(t *testing.T) {
	ok := okay.New[any]()

	ctx := context.Background()
	res := okay.Check[any](ctx, "hello", ok)

	assert := assert.New(t)
	assert.False(res.OK())
	assert.ErrorIs(res.Err(), okay.NotAllowed)
}

func TestOkay(t *testing.T) {
	type path string

	ok := okay.New[path]()
	ok.Add(func(_ context.Context, s path) okay.Response {
		return okay.Allow(true)
	})

	ok.Add(func(_ context.Context, s path) okay.Response {
		return okay.Errorf("bad request")
	})

	ctx := context.Background()
	res := ok.Allows(ctx, "/path/to/resource")

	assert := assert.New(t)
	assert.False(res.OK())
	assert.NotNil(res.Err())
	assert.Equal("bad request", res.Err().Error())
}
