package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestCache(t *testing.T) {
	c := cache.New(newClient(t))

	t.Run("empty", func(t *testing.T) {
		key := t.Name()

		value, err := c.Load(ctx, key)
		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
		is.Empty(value)
	})

	t.Run("exist", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		err := c.Store(ctx, key, value, time.Second)

		is := assert.New(t)
		is.Nil(err)

		loaded, err := c.Load(ctx, key)
		is.Nil(err)
		is.Equal(value, loaded)
	})

	t.Run("load many", func(t *testing.T) {
		prefix := t.Name()
		value := "hello"

		is := assert.New(t)

		var keys []string
		for i := range 3 {
			key := fmt.Sprintf("%s:%d", prefix, i)
			keys = append(keys, key)
			err := c.Store(ctx, key, value, time.Second)
			is.Nil(err)
		}

		unk := "not-exist"
		m, err := c.LoadMany(ctx, append(keys, unk)...)
		is.Nil(err)
		is.Len(m, 3)

		for k, v := range m {
			is.Equal(value, v)
			is.Contains(keys, k)
		}

		_, ok := m[unk]
		is.False(ok)
	})

	t.Run("load or store empty", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		old, loaded, err := c.LoadOrStore(ctx, key, value, time.Second)

		is := assert.New(t)
		is.Nil(err)
		is.Equal(value, old)
		is.False(loaded)

		old, loaded, err = c.LoadOrStore(ctx, key, value, time.Second)
		is.Nil(err)
		is.Equal(value, old)
		is.True(loaded)
	})

	t.Run("load or store exist", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		old, loaded, err := c.LoadOrStore(ctx, key, value, time.Second)

		is.Nil(err)
		is.Equal(value, old)
		is.True(loaded)
	})

	t.Run("load and delete empty", func(t *testing.T) {
		key := t.Name()
		old, loaded, err := c.LoadAndDelete(ctx, key)

		is := assert.New(t)
		is.Nil(err)
		is.Empty(old)
		is.False(loaded)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("load and delete exist", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		old, loaded, err := c.LoadAndDelete(ctx, key)

		is.Nil(err)
		is.Equal(value, old)
		is.True(loaded)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete empty", func(t *testing.T) {
		key := t.Name()
		old := "hello"
		deleted, err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.Nil(err)
		is.False(deleted)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete exist", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		deleted, err := c.CompareAndDelete(ctx, key, value)

		is.Nil(err)
		is.True(deleted)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap empty", func(t *testing.T) {
		key := t.Name()
		old := "hello"
		value := "hello"
		swapped, err := c.CompareAndSwap(ctx, key, old, value, time.Second)

		is := assert.New(t)
		is.Nil(err)
		is.False(swapped)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap exist", func(t *testing.T) {
		key := t.Name()
		value := "hello"

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		newValue := "world"
		swapped, err := c.CompareAndSwap(ctx, key, value, newValue, time.Second)

		is.Nil(err)
		is.True(swapped)

		loaded, err := c.Load(ctx, key)
		is.Nil(err)
		is.Equal(newValue, loaded)
	})
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
