package singleflight_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alextanhongpin/core/dsync/singleflight"
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

type User struct {
	Name string `json:"name"`
}

var john = &User{
	Name: "John",
}

func TestSingleflight(t *testing.T) {
	sf := singleflight.New[*User](newClient(t))
	v, err, shared := sf.Do(ctx, t.Name(), func(ctx context.Context) (*User, error) {
		return john, nil
	})
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal(john, v)

	v, err, shared = sf.Do(ctx, t.Name(), func(ctx context.Context) (*User, error) {
		return john, nil
	})
	is.Nil(err)
	is.True(shared)
	is.Equal(john, v)
}

func TestSingleflightConcurrent(t *testing.T) {
	sf := singleflight.New[*User](newClient(t))
	is := assert.New(t)

	counter := new(atomic.Int64)
	n := 10
	var wg sync.WaitGroup
	wg.Add(n)

	for _ = range n {
		go func() {
			defer wg.Done()

			v, err, shared := sf.Do(ctx, t.Name(), func(ctx context.Context) (*User, error) {
				return john, nil
			})
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
			is.Equal(john, v)

		}()
	}

	wg.Wait()
	is.Equal(int64(n-1), counter.Load())
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
