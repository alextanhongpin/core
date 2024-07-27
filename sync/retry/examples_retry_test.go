package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Error() {
	n := 10
	r := retry.New(n)
	r.Policy = func(ctx context.Context, i int) time.Duration {
		return time.Millisecond
	}

	i := 0
	start := time.Now()
	err := r.Do(func() error {
		i++
		return errors.New("random")
	})

	fmt.Println(err)
	fmt.Println(time.Now().Sub(start) > time.Duration(n)*time.Millisecond)
	fmt.Println(i)
	// Output:
	// retry: limit exceeded
	// random
	// true
	// 10
}
