package cache_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Married bool   `json:"married"`
}

var (
	john = &User{
		Name:    "John",
		Age:     30,
		Married: true,
	}
	jane = &User{
		Name:    "Jane",
		Age:     13,
		Married: false,
	}
)

func TestStruct(t *testing.T) {
	c := cache.NewStruct[*User](cache.New(newClient(t)))

	t.Run("empty", func(t *testing.T) {
		key := t.Name()

		value, err := c.Load(ctx, key)
		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
		is.Nil(value)
	})

	t.Run("exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)

		is := assert.New(t)
		is.Nil(err)

		loaded, err := c.Load(ctx, key)
		is.Nil(err)
		is.Equal(value, loaded)
	})

	t.Run("load or store empty", func(t *testing.T) {
		key := t.Name()
		value := john

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
		value := john

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
		is.Nil(old)
		is.False(loaded)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("load and delete exist", func(t *testing.T) {
		key := t.Name()
		value := john

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
		old := john
		deleted, err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.Nil(err)
		is.False(deleted)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete exist", func(t *testing.T) {
		key := t.Name()
		value := john

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
		old := john
		value := jane
		swapped, err := c.CompareAndSwap(ctx, key, old, value, time.Second)

		is := assert.New(t)
		is.Nil(err)
		is.False(swapped)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		newValue := jane
		swapped, err := c.CompareAndSwap(ctx, key, value, newValue, time.Second)

		is.Nil(err)
		is.True(swapped)

		loaded, err := c.Load(ctx, key)
		is.Nil(err)
		is.Equal(newValue, loaded)
	})
}
