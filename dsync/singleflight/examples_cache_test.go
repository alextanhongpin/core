package singleflight_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func ExampleCache() {
	// Cache example.
	// When the cache is stale, only once worker will populate
	// the cache, and the rest will wait.

	stop := redistest.Init()
	defer stop()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	defer client.Close()

	cache := singleflight.NewCache[string](client)
	ctx := context.Background()

	hit := new(atomic.Int64)
	exec := new(atomic.Int64)

	n := 10
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			v, hitOrMiss, err := cache.LoadOrStore(ctx, "foo", func(context.Context) (string, error) {
				exec.Add(1)
				return "bar", nil
			}, 10*time.Second)
			if err != nil {
				panic(err)
			}
			if v != "bar" {
				panic("unexpected value: " + v)
			}
			if hitOrMiss {
				hit.Add(1)
			}

		}()
	}
	wg.Wait()

	fmt.Println("hit:", hit.Load())
	fmt.Println("exec:", exec.Load())

	// Output:
	// hit: 9
	// exec: 1
}
