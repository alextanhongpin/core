package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	redis "github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	//_ = rdb.FlushDB(ctx).Err()
	defer rdb.Close()

	rl, err := ratelimit.New(rdb, &ratelimit.Option{
		Limit: 5,
		Every: 2 * time.Second,
		Burst: 1,
	})
	if err != nil {
		panic(err)
	}

	key := "user:1"
	now := time.Now()
	limit := 0
	for i := 0; i < 30; i++ {
		res, err := rl.SlidingWindow(ctx, key)
		if err != nil {
			log.Fatalf("got error: %v", err)
		}

		if res.Allow {
			limit++
		}
		fmt.Printf("%#v\n", res)
		fmt.Println(res.RetryIn, res.ResetIn)
		fmt.Println()

		if time.Since(now) >= 3*time.Second {
			break
		}
		sleep := time.Duration(rand.Intn(100))
		time.Sleep(sleep * time.Millisecond)
	}

	fmt.Println("total", limit, "took", time.Since(now))
}
