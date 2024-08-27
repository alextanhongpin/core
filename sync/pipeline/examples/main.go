package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
	"golang.org/x/exp/rand"
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
	p2 = pipeline.Rate(p2, func(rate pipeline.RateInfo) {
		if rate.Total%10 == 0 {
			fmt.Println(rate)
		}
	})
	p2 = pipeline.Queue(100, p2)
	p3 := pipeline.Pool(5, p2, func(v string) string {
		time.Sleep(100 * time.Millisecond)
		return v
	})

	p4 := pipeline.Map(p3, func(v string) pipeline.Result[string] {
		if rand.Intn(10) < 7 {
			return pipeline.Result[string]{Data: v}
		}

		return pipeline.Result[string]{Err: fmt.Errorf("error")}
	})

	p4 = pipeline.Throughput(p4, func(t pipeline.ThroughputInfo) {
		fmt.Println(t)
	})

	p5 := pipeline.FlatMap(p4)

	stop := pipeline.Batch(3, time.Second, p5, func(in []string) {
		fmt.Println(in)
	})
	defer stop()
}
