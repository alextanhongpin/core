package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

func TestFunc_Gob(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	cacheDir := fmt.Sprintf("testdata/%s/.cache", t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(cacheDir)
	})
	fn := cache.Func(func(ctx context.Context, v string) (data, error) {
		count++
		return data{Value: v}, nil
	}, &cache.FuncConfig{
		CacheDir: cacheDir,
		Codec:    cache.NewGobCodec(),
	})

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

func TestFunc_JSON(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	cacheDir := fmt.Sprintf("testdata/%s/.cache", t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(cacheDir)
	})
	fn := cache.Func(func(ctx context.Context, v string) (data, error) {
		count++
		return data{Value: v}, nil
	}, &cache.FuncConfig{
		CacheDir: cacheDir,
		Codec:    cache.NewJSONCodec(),
	})

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

func TestFunc2_Gob(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	storage, err := cache.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fn := cache.Func2(func(ctx context.Context, v string) (data, error) {
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

func TestFunc2_JSON(t *testing.T) {
	type data struct {
		Value string `json:"value"`
	}
	var count int
	storage, err := cache.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fn := cache.Func2(func(ctx context.Context, v string) (data, error) {
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
