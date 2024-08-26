package promise_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func ExamplePromiseNew() {
	counter := new(atomic.Int64)
	g := promise.NewGroup[int]()
	n := 10

	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()

			v, err := g.DoAndForget("key", func() (int, error) {
				counter.Add(1)
				time.Sleep(100 * time.Millisecond)
				return 42, nil
			})
			fmt.Println(v, err)
		}()
	}

	wg.Wait()
	fmt.Println("called", counter.Load())
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
