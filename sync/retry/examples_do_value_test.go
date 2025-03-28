package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleDoValue() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
	defer cancel()

	// Success case.
	n, err := retry.DoValue(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	}, 3)
	fmt.Println(n)
	fmt.Println(err)

	// Timeout case.
	_, err = retry.DoValue(ctx, func(ctx context.Context) (int, error) {
		return 0, errors.ErrUnsupported
	}, 3)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, context.DeadlineExceeded))

	// Limit exceeded case.
	_, err = retry.DoValue(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errors.ErrUnsupported
	}, 1)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, retry.ErrLimitExceeded))

	// Output:
	// 42
	// <nil>
	// context deadline exceeded
	// unsupported operation
	// true
	// true
	// retry: limit exceeded
	// unsupported operation
	// true
	// true
}
