package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	r := retry.New(retry.NoWait, retry.Throttle(), retry.N(3))

	var mu sync.Mutex
	counter := make(map[int]int)

	ctx := context.Background()

	var failed, success int64
	var skipped int64
	n := 100

	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			var i int
			err := r.Do(ctx, func(context.Context) error {
				i++

				mu.Lock()
				counter[i]++
				mu.Unlock()

				if rand.Float64() < 0.2 {
					return errors.ErrUnsupported
				}

				return nil
			})

			if errors.Is(err, retry.ErrThrottled) {
				skipped++
			}
			if errors.Is(err, retry.ErrLimitExceeded) {
				failed++
			}
			if err == nil {
				success++
			}
		})
	}

	wg.Wait()
	fmt.Println("success:", success)
	fmt.Println("skipped:", skipped)
	fmt.Println("failed:", failed)
	fmt.Println("counter:", counter)
	var retries int
	for k, v := range counter {
		if k > 0 {
			retries += v
		}
	}
	fmt.Println("retries:", retries)
}
