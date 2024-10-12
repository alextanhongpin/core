package cache_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	c := cache.NewJSON(cache.New(newClient(t)))

	t.Run("empty", func(t *testing.T) {
		key := t.Name()

		var value *User
		err := c.Load(ctx, key, &value)
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

		var loaded *User
		err = c.Load(ctx, key, &loaded)
		is.Nil(err)
		is.Equal(value, loaded)
	})

	t.Run("load and delete empty", func(t *testing.T) {
		key := t.Name()
		var old *User
		loaded, err := c.LoadAndDelete(ctx, key, &old)

		is := assert.New(t)
		is.Nil(err)
		is.Nil(old)
		is.False(loaded)

		err = c.Load(ctx, key, new(User))
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("load and delete exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.Nil(err)

		var old *User
		loaded, err := c.LoadAndDelete(ctx, key, &old)

		is.Nil(err)
		is.Equal(value, old)
		is.True(loaded)

		err = c.Load(ctx, key, new(User))
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete empty", func(t *testing.T) {
		key := t.Name()
		old := john
		deleted, err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.Nil(err)
		is.False(deleted)

		err = c.Load(ctx, key, new(User))
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

		err = c.Load(ctx, key, new(User))
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

		err = c.Load(ctx, key, new(User))
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

		var loaded *User
		err = c.Load(ctx, key, &loaded)
		is.Nil(err)
		is.Equal(newValue, loaded)
	})
}
