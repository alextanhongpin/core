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
	r := retry.New(retry.NewConstantBackOff(0))
	r.Throttler = retry.NewThrottler(retry.NewThrottlerOptions())

	var mu sync.Mutex
	counter := make(map[int]int)

	ctx := context.Background()

	var failed, success int64
	var skipped int64
	n := 100

	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()

			ratio := int64(2)
			fn := func() error {
				if rand.Int64N(10) < ratio {
					return errors.New("failed")
				}

				return nil
			}

			for i, err := range r.Try(ctx, 10) {
				mu.Lock()
				counter[i]++
				mu.Unlock()

				if err != nil {
					if errors.Is(err, retry.ErrThrottled) {
						skipped++
					}
					if errors.Is(err, retry.ErrLimitExceeded) {
						failed++
					}
					break
				}

				if err := fn(); err == nil {
					success++
					break
				}

			}
		}()
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
