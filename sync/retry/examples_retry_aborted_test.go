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

	r := retry.New(10)
	r.Policy = func(ctx context.Context, i int) time.Duration {
		return time.Millisecond
	}

	err := r.DoCtx(ctx, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			// Cancel retry when timeout.
			return errors.Join(retry.ErrAborted, context.Cause(ctx))
		default:
			return errors.New("random")
		}
	})

	fmt.Println(err)
	// Output:
	// retry: aborted
	// context deadline exceeded
}
