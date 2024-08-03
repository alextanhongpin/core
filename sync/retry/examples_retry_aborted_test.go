package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Abort() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	opts := retry.NewOptions()
	opts.Policy = func(i int) time.Duration {
		return time.Millisecond
	}
	r := retry.New(opts)

	err := r.Do(func() error {
		select {
		case <-ctx.Done():
			// Cancel retry when timeout.
			return retry.Abort(context.Cause(ctx))
		default:
			return errors.New("random")
		}
	})

	fmt.Println(err)
	fmt.Println(errors.Unwrap(err))
	// Output:
	// retry: aborted: context deadline exceeded
	// context deadline exceeded
}
