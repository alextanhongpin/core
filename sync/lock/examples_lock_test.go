package lock_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

func ExampleLock() {
	locker := lock.New()

	start := time.Now()
	defer func(start time.Time) {
		fmt.Println("Total time taken less than 510ms:", time.Since(start) < 510*time.Millisecond)
	}(start)

	n := 10
	debug := false

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i%2)

			l := locker.Get(key)

			if debug {
				fmt.Println("acquiring lock...", key, time.Since(start))
			}
			l.Lock()
			defer l.Unlock()

			time.Sleep(100 * time.Millisecond)
			if debug {
				fmt.Println("releasing lock...", key, time.Since(start))
			}
		}()
	}

	wg.Wait()
	fmt.Println("exiting...")
	// Output:
	// Total time taken less than 510ms: true
}
