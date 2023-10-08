package main

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
	rl := ratelimit.NewMultiFixedWindow(
		ratelimit.Rate{Limit: 5, Period: 1 * time.Second},
		ratelimit.Rate{Limit: 15, Period: 5 * time.Second},
		ratelimit.Rate{Limit: 30, Period: 10 * time.Second},
	)
	//rl := ratelimit.NewFixedWindow(5, time.Second)
	//rl := ratelimit.NewLeakyBucket(10, time.Second, 0)
	now := time.Now()
	var count int
	for time.Since(now) < 10*time.Second {
		result := rl.Allow()
		sleep := 75 * time.Millisecond
		//sleep := time.Duration(rand.Intn(100)+50) * time.Millisecond
		fmt.Println(time.Since(now), result.Allow, result.RetryIn(), result.Remaining)

		if result.Allow {
			count++
		}
		time.Sleep(sleep)
	}
	fmt.Println("total", count)
}
