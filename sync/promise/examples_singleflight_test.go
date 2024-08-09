package promise_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func ExamplePromiseNew() {
	count := new(atomic.Int64)
	var p *promise.Promise[int]

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)

	var mu sync.Mutex

	for range n {
		go func() {
			defer wg.Done()

			var local *promise.Promise[int]
			mu.Lock()
			if p == nil {
				p = promise.New(func() (int, error) {
					count.Add(1)
					time.Sleep(100 * time.Millisecond)
					return 42, nil
				})
			}
			local = p
			mu.Unlock()

			fmt.Println(local.Await())
		}()
	}

	wg.Wait()
	fmt.Println("called", count.Load())
	// Output:
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// 42 <nil>
	// called 1
}
