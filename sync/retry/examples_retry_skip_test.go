package retry_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetrySkip() {
	ctx := context.Background()

	var i int
	errs := []error{
		errors.New("random"),
		errors.New("random"),
		retry.Skip(errors.New("random")),
	}

	backoffs := retry.Backoffs{0, 0, 0, 0, 0}
	res, err := backoffs.ExecResult(ctx, func(ctx context.Context) error {
		err := errs[i]
		i++
		return err
	})
	fmt.Println(err)
	fmt.Println(res.Retry)
	fmt.Println(res.Skip)
	// Output:
	// random
	// 2
	// true
}
