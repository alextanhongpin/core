package cache_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

func testStorage(t *testing.T, c cache.Storage[[]byte]) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		key := t.Name()

		value, err := c.Load(ctx, key)
		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
		is.Empty(value)
	})

	t.Run("exist", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.Store(ctx, key, value, time.Second)

		is := assert.New(t)
		is.NoError(err)

		loaded, err := c.Load(ctx, key)
		is.NoError(err)
		is.Equal(value, loaded)
	})

	t.Run("store once", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.StoreOnce(ctx, key, value, time.Second)

		is := assert.New(t)
		is.NoError(err)

		err = c.StoreOnce(ctx, key, value, time.Second)
		is.ErrorIs(err, cache.ErrExists)
	})

	t.Run("load or store empty", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		old, loaded, err := c.LoadOrStore(ctx, key, value, time.Second)

		is := assert.New(t)
		is.NoError(err)
		is.Equal(value, old)
		is.False(loaded)

		old, loaded, err = c.LoadOrStore(ctx, key, value, time.Second)
		is.NoError(err)
		is.Equal(value, old)
		is.True(loaded)
	})

	t.Run("load or store exist", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		old, loaded, err := c.LoadOrStore(ctx, key, value, time.Second)

		is.NoError(err)
		is.Equal(value, old)
		is.True(loaded)
	})

	t.Run("load and delete empty", func(t *testing.T) {
		key := t.Name()
		old, err := c.LoadAndDelete(ctx, key)
		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
		is.Empty(old)
	})

	t.Run("load and delete exist", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		old, err := c.LoadAndDelete(ctx, key)
		is.NoError(err)
		is.Equal(value, old)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete empty", func(t *testing.T) {
		key := t.Name()
		old := []byte("hello")
		err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete exist", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		err = c.CompareAndDelete(ctx, key, value)
		is.NoError(err)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap empty", func(t *testing.T) {
		key := t.Name()
		old := []byte("hello")
		value := []byte("hello")
		err := c.CompareAndSwap(ctx, key, old, value, time.Second)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap exist", func(t *testing.T) {
		key := t.Name()
		value := []byte("hello")

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		newValue := []byte("world")
		err = c.CompareAndSwap(ctx, key, value, newValue, time.Second)
		is.NoError(err)

		loaded, err := c.Load(ctx, key)
		is.NoError(err)
		is.Equal(newValue, loaded)
	})
}
