package cache_test

import (
	"context"
	"os"
	"sync"
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
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	c := cache.New(newClient(t), func(ctx context.Context) (*User, error) {
		return &User{
			Name: "John Appleseed",
			Age:  13,
		}, nil
	})

	is := assert.New(t)
	u, hit, err := c.Get(ctx, "john.appleseed", 5*time.Second)
	is.Nil(err)
	is.False(hit)
	is.Equal("John Appleseed", u.Name)
	is.Equal(13, u.Age)

	_, hit, err = c.Get(ctx, "john.appleseed", 5*time.Second)
	is.Nil(err)
	is.True(hit)
}

func TestSingleflight(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	c := cache.New(newClient(t), func(ctx context.Context) (*User, error) {
		time.Sleep(100 * time.Millisecond)
		return &User{
			Name: "John Appleseed",
			Age:  13,
		}, nil
	})
	c.SingleFlight = &cache.SingleFlight{
		Retries: []time.Duration{100 * time.Millisecond, 25 * time.Millisecond},
		KeyFn: func(key string) string {
			return "singleflight:" + key
		},
	}

	is := assert.New(t)

	// Simulate 2 concurrent requests. Only, one will hit the cache.
	// The second will wait for the cache to be populated.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		u, hit, err := c.Get(ctx, "john.appleseed", 5*time.Second)
		is.Nil(err)
		is.False(hit)
		is.Equal("John Appleseed", u.Name)
		is.Equal(13, u.Age)
	}()

	go func() {
		defer wg.Done()

		time.Sleep(5 * time.Millisecond)
		start := time.Now()
		u, hit, err := c.Get(ctx, "john.appleseed", 5*time.Second)
		is.Nil(err)
		is.True(hit)
		is.Equal("John Appleseed", u.Name)
		is.Equal(13, u.Age)
		is.Greater(time.Since(start), c.SingleFlight.Retries[0])
		is.Less(time.Since(start), c.SingleFlight.Retries[0]+c.SingleFlight.Retries[1])
	}()

	wg.Wait()
}

func TestWithoutSingleflight(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	c := cache.New(newClient(t), func(ctx context.Context) (*User, error) {
		time.Sleep(100 * time.Millisecond)
		return &User{
			Name: "John Appleseed",
			Age:  13,
		}, nil
	})

	is := assert.New(t)

	// Simulate 2 concurrent requests. Only, both will hit the cache.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		u, hit, err := c.Get(ctx, "john.appleseed", 5*time.Second)
		is.Nil(err)
		is.False(hit)
		is.Equal("John Appleseed", u.Name)
		is.Equal(13, u.Age)
	}()

	go func() {
		defer wg.Done()

		u, hit, err := c.Get(ctx, "john.appleseed", 5*time.Second)
		is.Nil(err)
		is.False(hit)
		is.Equal("John Appleseed", u.Name)
		is.Equal(13, u.Age)
	}()

	wg.Wait()
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
