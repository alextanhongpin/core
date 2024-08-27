package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	now := time.Now()
	defer func() {
		fmt.Println(time.Since(now))
	}()

	p1 := pipeline.RateLimit(60, time.Second, pipeline.Generator(ctx, 100))

	p2 := pipeline.Map(p1, func(i int) string {
		return strconv.Itoa(i)
	})
	p2 = pipeline.Progress(time.Second, p2, func(total int, rate int64) {
		if total%10 == 0 {
			fmt.Printf("Count: %d, %dreq/s\n", total, rate)
		}
	})
	p2 = pipeline.Queue(100, p2)
	p3 := pipeline.Pool(5, p2, func(v string) string {
		time.Sleep(100 * time.Millisecond)
		return v
	})
	stop := pipeline.Batch(3, time.Second, p3, func(in []string) {
		fmt.Println(in)
	})
	defer stop()
}
