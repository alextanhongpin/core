package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	opts := retry.NewOptions()
	//opts.Throttler = nil

	var mu sync.Mutex
	counter := make(map[int]int)
	opts.Policy = func(i int) time.Duration {
		mu.Lock()
		counter[i]++
		mu.Unlock()
		return 0
	}
	r := retry.New(opts)

	var failed, success int64
	var skipped int64
	n := 100

	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()

			i := rand.Float64()
			err := r.Do(func() error {
				if rand.Float64() < i {
					return errors.New("failed")
				}
				return nil
			})
			if err != nil {
				if errors.Is(err, retry.ErrThrottled) {
					skipped++
					fmt.Println("skipped")
				}
				failed++
			} else {
				success++
				fmt.Println("ok")
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
