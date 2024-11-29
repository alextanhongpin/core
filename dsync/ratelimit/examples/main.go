package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

func main() {
	stop := redistest.Init()
	defer stop()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	_ = rdb.FlushDB(ctx).Err()
	defer rdb.Close()

	{
		key := "user:1"
		rl := newFixedWindow(rdb)
		if err := simulate(ctx, rl, key); err != nil {
			panic(err)
		}
	}
	{
		key := "user:2"
		rl := newGCRA(rdb)
		if err := simulate(ctx, rl, key); err != nil {
			panic(err)
		}
	}

	fmt.Println("end")
}

type ratelimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

func simulate(ctx context.Context, rl ratelimiter, key string) error {
	now := time.Now()
	fmt.Println("start", now)
	limit := 0

	for time.Since(now) < 1*time.Second {
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			return fmt.Errorf("allow error: %w", err)
		}

		if allow {
			limit++
		}

		fmt.Printf("elapsed: %s, allow=%t\n", time.Since(now), allow)
		sleep := time.Duration(rand.Intn(100))
		time.Sleep(sleep * time.Millisecond)
	}

	fmt.Println("total", limit, "took", time.Since(now))
	return nil
}

func newFixedWindow(client *redis.Client) *ratelimit.FixedWindow {
	return ratelimit.NewFixedWindow(client, 5, time.Second)
}

func newGCRA(client *redis.Client) *ratelimit.GCRA {
	return ratelimit.NewGCRA(client, 5, time.Second, 1)
}
