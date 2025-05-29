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

func TestJSON(t *testing.T) {
	c := cache.NewJSON(newClient(t))

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
		is.NoError(err)

		var loaded *User
		err = c.Load(ctx, key, &loaded)
		is.NoError(err)
		is.Equal(value, loaded)
	})

	t.Run("load and delete empty", func(t *testing.T) {
		key := t.Name()
		var old *User
		err := c.LoadAndDelete(ctx, key, &old)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
		is.Nil(old)
	})

	t.Run("load and delete exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		var old *User
		err = c.LoadAndDelete(ctx, key, &old)

		is.NoError(err)
		is.Equal(value, old)

		err = c.Load(ctx, key, new(User))
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete empty", func(t *testing.T) {
		key := t.Name()
		old := john
		err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)

		err = c.Load(ctx, key, new(User))
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		err = c.CompareAndDelete(ctx, key, value)
		is.NoError(err)

		err = c.Load(ctx, key, new(User))
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap empty", func(t *testing.T) {
		key := t.Name()
		old := john
		value := jane
		err := c.CompareAndSwap(ctx, key, old, value, time.Second)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and swap exist", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.Store(ctx, key, value, time.Second)
		is := assert.New(t)
		is.NoError(err)

		newValue := jane
		err = c.CompareAndSwap(ctx, key, value, newValue, time.Second)
		is.NoError(err)

		var loaded *User
		err = c.Load(ctx, key, &loaded)
		is.NoError(err)
		is.Equal(newValue, loaded)
	})

	t.Run("load or store", func(t *testing.T) {
		key := t.Name()
		is := assert.New(t)
		getter := func() (any, error) {
			return john, nil
		}

		var u *User
		loaded, err := c.LoadOrStore(ctx, key, &u, getter, time.Minute)
		is.NoError(err)
		is.False(loaded)

		var v *User
		loaded, err = c.LoadOrStore(ctx, key, &v, getter, time.Minute)
		is.NoError(err)
		is.True(loaded)
		is.Equal(*u, *v)
	})
}
