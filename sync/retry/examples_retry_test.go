package retry_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry() {
	opt := retry.NewOption()
	opt.Delay = 0
	r := retry.New(opt)

	ctx := context.Background()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		return errors.New("random")
	})
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// retry.Result: <nil>
	// retry: max attempts reached
	// random
}
