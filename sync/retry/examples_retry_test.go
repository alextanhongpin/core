package retry_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Error() {
	opts := retry.NewOptions()
	opts.Attempts = 10
	opts.Policy = func(i int) time.Duration {
		return time.Millisecond
	}
	r := retry.New(opts)

	var wantErr = errors.New("random")
	i := 0
	start := time.Now()
	err := r.Do(func() error {
		i++
		return wantErr
	})

	fmt.Println(err)
	fmt.Println(errors.Is(err, retry.ErrLimitExceeded))
	fmt.Println(errors.Is(err, wantErr))
	fmt.Println(errors.Unwrap(err))
	fmt.Println(time.Since(start) > time.Duration(opts.Attempts)*time.Millisecond)
	fmt.Println(i)
	// Output:
	// retry: aborted: retry: throttled
	// false
	// false
	// retry: throttled
	// false
	// 5
}
