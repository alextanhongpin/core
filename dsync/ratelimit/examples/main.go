package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	redis "github.com/redis/go-redis/v9"
)

func init() {
	rand.Seed(42)
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	_ = rdb.FlushDB(ctx).Err()
	defer rdb.Close()

	key := "user:1"
	rl := newFixedWindow(rdb)
	if err := simulate(ctx, rl, key, 30); err != nil {
		panic(err)
	}

	sw := newSlidingWindow(rdb)
	key = "random:2"
	if err := simulateSlidingWindow(ctx, sw, key, 30); err != nil {
		panic(err)
	}

	fmt.Println("end")
}

type ratelimiter interface {
	Allow(ctx context.Context, key string) (*ratelimit.Result, error)
}

func simulate(ctx context.Context, rl ratelimiter, key string, n int) error {
	now := time.Now()
	fmt.Println("start", now)
	limit := 0
	for i := 0; i < n; i++ {
		res, err := rl.Allow(ctx, key)
		if err != nil {
			return err
		}

		if res.Allow {
			limit++
		}

		fmt.Printf("elapsed: %s, %#v\n", time.Since(now), res)
		sleep := time.Duration(rand.Intn(100))
		time.Sleep(sleep * time.Millisecond)
	}

	fmt.Println("total", limit, "took", time.Since(now))
	return nil
}

func newGCRA(client *redis.Client) *ratelimit.GCRA {
	return ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  0,
	})
}

func newFixedWindow(client *redis.Client) *ratelimit.FixedWindow {
	return ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})
}

func newSlidingWindow(client *redis.Client) *ratelimit.SlidingWindow {
	return ratelimit.NewSlidingWindow(client, &ratelimit.SlidingWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})
}

type slidingWindowRateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

func simulateSlidingWindow(ctx context.Context, rl slidingWindowRateLimiter, key string, n int) error {
	now := time.Now()
	fmt.Println("start", now)
	limit := 0
	for i := 0; i < n; i++ {
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			return err
		}

		if allow {
			limit++
		}

		fmt.Printf("elapsed: %s, %#v\n", time.Since(now), allow)
		sleep := time.Duration(rand.Intn(100))
		time.Sleep(sleep * time.Millisecond)
	}

	fmt.Println("total", limit, "took", time.Since(now))
	return nil
}
