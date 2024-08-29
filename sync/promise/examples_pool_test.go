package promise_test

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func ExamplePool() {
	var total int
	now := time.Now()
	pool := promise.NewPool[int](1)
	for range 3 {
		pool.Do(func() (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 42, nil
		})
	}
	vs, err := pool.All()
	if err != nil {
		panic(err)
	}
	for _, v := range vs {
		total += v
	}

	fmt.Println(">30ms", time.Since(now) > 30*time.Millisecond)
	now = time.Now()
	pool = promise.NewPool[int](3)
	for range 3 {
		pool.Do(func() (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 42, nil
		})
	}
	vs, err = pool.All()
	if err != nil {
		panic(err)
	}
	for _, v := range vs {
		total += v
	}
	fmt.Println(">10ms", time.Since(now) > 10*time.Millisecond && time.Since(now) < 15*time.Millisecond)
	fmt.Println("total:", total)
	// Output:
	// >30ms true
	// >10ms true
	// total: 252
}
