package cache_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

func TestGoFunc_Gob(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	storage, err := cache.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fn := cache.GoFunc(func(ctx context.Context, v string) (data, error) {
		count++
		return data{Value: v}, nil
	},
		storage,
		cache.NewGobCodec(),
	)

	ctx := t.Context()
	args := t.Name()

	is := assert.New(t)
	for range 3 {
		res, err := fn(ctx, args)
		is.NoError(err)
		is.Equal(t.Name(), res.Value)
		is.Equal(1, count)
	}
}

func TestGoFunc_JSON(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	storage, err := cache.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fn := cache.GoFunc(func(ctx context.Context, v string) (data, error) {
		count++
		return data{Value: v}, nil
	},
		storage,
		cache.NewJSONCodec(),
	)

	ctx := t.Context()
	args := t.Name()

	is := assert.New(t)
	for range 3 {
		res, err := fn(ctx, args)
		is.NoError(err)
		is.Equal(t.Name(), res.Value)
		is.Equal(1, count)
	}
}
