package retry_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetryDo() {
	r := retry.New(nil)
	ctx := context.Background()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		return nil
	})
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// retry.Result: retry 0 times, took 0s
	// <nil>
}
