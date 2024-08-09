package promise_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func ExamplePromiseDeferred() {
	now := time.Now()
	p := promise.Deferred[int]()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("...waiting")
		fmt.Println(p.Await())
		fmt.Println(">100ms", time.Since(now) > 100*time.Millisecond)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(100 * time.Millisecond)
		p.Resolve(42)
	}()

	wg.Wait()
	// Output:
	// ...waiting
	// 42 <nil>
	// >100ms true
}
