package backpressure_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/exp/backpressure"
)

func ExampleNew() {
	g := backpressure.New(1)
	defer g.Flush()

	race := make(chan bool)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		<-race

		// Start slower than the other goroutine.
		time.Sleep(25 * time.Millisecond)

		// Attempt to acquire the lock within 50ms.
		// Otherwise, drop the request.
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := g.Lock(ctx); err != nil {
			// Backpressure applied.
			fmt.Println(errors.Is(err, context.Canceled))
			return
		}
	}()

	go func() {
		defer wg.Done()

		<-race

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := g.Lock(ctx); err != nil {
			panic(err)
		}
		defer g.Unlock()

		// Simulate work that holds the lock for 100ms.
		// The other goroutine cannot continue until this is
		// completed.
		time.Sleep(100 * time.Millisecond)
		fmt.Println("acquired")
	}()

	// Start both goroutine.
	close(race)

	wg.Wait()

	// Output:
	// false
	// acquired
}
