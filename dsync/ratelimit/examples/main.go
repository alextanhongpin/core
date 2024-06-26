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
	//rl := newFixedWindow(rdb)
	rl := newTokenBucket(rdb)
	if err := simulate(ctx, rl, key, 30); err != nil {
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

	start := now.Truncate(time.Minute).Add(time.Minute)
	wait := start.Sub(now)
	fmt.Println("wait for", wait)
	time.Sleep(wait)

	now = time.Now()

	for time.Since(now) < 1*time.Second {
		res, err := rl.Allow(ctx, key)
		if err != nil {
			return fmt.Errorf("allow error: %w", err)
		}

		if res.Allow {
			limit++
		}

		fmt.Printf("elapsed: %s, allow=%t remaining=%d retryIn=%s resetIn=%s now=%s\n",
			time.Since(now),
			res.Allow,
			res.Remaining,
			res.RetryIn(),
			res.ResetIn(),
			time.Now().Format(time.DateTime),
		)
		sleep := time.Duration(rand.Intn(100))
		time.Sleep(sleep * time.Millisecond)
	}

	fmt.Println("total", limit, "took", time.Since(now))
	return nil
}

func newFixedWindow(client *redis.Client) *ratelimit.FixedWindow {
	return ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})
}

func newTokenBucket(client *redis.Client) *ratelimit.TokenBucket {
	return ratelimit.NewTokenBucket(client, &ratelimit.TokenBucketOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  3,
	})
}
