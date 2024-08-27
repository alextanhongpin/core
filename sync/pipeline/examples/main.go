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
	p2 = pipeline.Rate(time.Second, p2, func(rate pipeline.RateInfo) {
		if rate.Total%10 == 0 {
			fmt.Println(rate)
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
