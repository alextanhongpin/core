package cache_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
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

func TestRedisJSON(t *testing.T) {
	c := cache.New[*User]()
	c.Storage = cache.NewRedis(newClient(t))

	testUnmarshaler(t, c)
}

func TestRedisGob(t *testing.T) {
	c := cache.New[*User]()
	c.Storage = cache.NewRedis(newClient(t))
	c.Encoder = cache.NewGobEncoder()

	testUnmarshaler(t, c)
}

func TestFileJSON(t *testing.T) {
	path := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(path))
	})
	storage, err := cache.NewFile(path)
	assert.NoError(t, err)
	c := cache.New[*User]()
	c.Storage = storage
	assert.NoError(t, err)

	testUnmarshaler(t, c)
}

func TestFileGob(t *testing.T) {
	path := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(path))
	})
	storage, err := cache.NewFile(path)
	assert.NoError(t, err)

	c := cache.New[*User]()
	c.Storage = storage
	c.Encoder = cache.NewGobEncoder()

	testUnmarshaler(t, c)
}

func testUnmarshaler(t *testing.T, c cache.Storage[*User]) {
	t.Helper()
	t.Cleanup(func() {
		_ = c.Close()
	})

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
		is.NoError(err)

		loaded, err := c.Load(ctx, key)
		is.NoError(err)

		is.Equal(value, loaded)
	})

	t.Run("load and delete empty", func(t *testing.T) {
		key := t.Name()
		old, err := c.LoadAndDelete(ctx, key)

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

		old, err := c.LoadAndDelete(ctx, key)

		is.NoError(err)
		is.Equal(value, old)

		_, err = c.Load(ctx, key)
		is.ErrorIs(err, cache.ErrNotExist)
	})

	t.Run("compare and delete empty", func(t *testing.T) {
		key := t.Name()
		old := john
		err := c.CompareAndDelete(ctx, key, old)

		is := assert.New(t)
		is.ErrorIs(err, cache.ErrNotExist)

		_, err = c.Load(ctx, key)
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

		_, err = c.Load(ctx, key)
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

		loaded, err := c.Load(ctx, key)
		is.NoError(err)
		is.Equal(newValue, loaded)
	})

	t.Run("load or store", func(t *testing.T) {
		sep := cache.Separator(':')
		key := sep.Join("users", t.Name())

		is := assert.New(t)
		getter := func(ctx context.Context, key string) (*User, time.Duration, error) {
			parts := sep.Split(key)
			t.Log("fetching", parts[1])
			return john, time.Minute, nil
		}

		u, loaded, err := c.LoadOrStoreFunc(ctx, key, getter)
		is.NoError(err)
		is.False(loaded)

		v, loaded, err := c.LoadOrStoreFunc(ctx, key, getter)
		is.NoError(err)
		is.True(loaded)
		is.Equal(u, v)
	})

	t.Run("load or store concurrent", func(t *testing.T) {
		key := t.Name()
		is := assert.New(t)

		var calls atomic.Int64
		getter := func(ctx context.Context, key string) (*User, time.Duration, error) {
			calls.Add(1)
			time.Sleep(100 * time.Millisecond)
			return john, time.Minute, nil
		}

		var counter atomic.Int64
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				u, loaded, err := c.LoadOrStoreFunc(ctx, key, getter)
				is.NoError(err)
				is.Equal(u, john)
				if loaded {
					counter.Add(1)
				}
			})
		}
		wg.Wait()

		is.Equal(int64(1), calls.Load())
		is.Equal(int64(9), counter.Load())
	})

	t.Run("store once", func(t *testing.T) {
		key := t.Name()
		value := john

		err := c.StoreOnce(ctx, key, value, time.Second)

		is := assert.New(t)
		is.NoError(err)

		err = c.StoreOnce(ctx, key, value, time.Second)
		is.ErrorIs(err, cache.ErrExists)
	})
}
