package singleflight_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func ExampleGroup() {
	// Cache example.
	// When the cache is stale, only once worker will populate
	// the cache, and the rest will wait.

	stop := redistest.Init()
	defer stop()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	defer client.Close()

	r := new(Repository)
	r.client = client
	r.group = &singleflight.Group{
		Client: client,
		Locker: lock.New(client),
	}

	ctx := context.Background()

	hit := new(atomic.Int64)
	exec := new(atomic.Int64)

	n := 10
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			v, hitOrMiss, err := r.Get(ctx, "foo", func(context.Context) (string, error) {
				exec.Add(1)
				return "bar", nil
			})
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

type Repository struct {
	group  *singleflight.Group
	client *redis.Client
}

func (r *Repository) Get(ctx context.Context, key string, getter func(ctx context.Context) (string, error)) (s string, hitOrMiss bool, err error) {
	lockTTL := 10 * time.Second
	waitTTL := 10 * time.Second

	s, err = r.client.Get(ctx, key).Result()
	if err == nil {
		return s, true, nil
	}
	if !errors.Is(err, redis.Nil) {
		return s, false, err
	}

	doOrWait, err := r.group.DoOrWait(ctx, key+":fetch", func(ctx context.Context) error {
		v, err := getter(ctx)
		if err != nil {
			return err
		}

		return r.client.Set(ctx, key, v, time.Second).Err()
	}, lockTTL, waitTTL)
	if err != nil {
		return "", false, err
	}

	s, err = r.client.Get(ctx, key).Result()
	if err != nil {
		return "", false, err
	}

	if doOrWait {
		// If did, then it's a miss.
		hitOrMiss = false
	} else {
		// If waited, then it's a hit
		hitOrMiss = true
	}

	return s, hitOrMiss, nil
}
