package retry_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry() {
	ctx := context.Background()
	backoffs := retry.Backoffs{0, 0, 0, 0, 0}
	res, err := backoffs.ExecResult(ctx, func(ctx context.Context) error {
		return errors.New("random")
	})
	fmt.Println(err)
	fmt.Println(res.Retry)
	fmt.Println(res.Skip)
	// Output:
	// random
	// 5
	// false
}
