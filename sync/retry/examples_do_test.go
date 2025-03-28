package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleDo() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
	defer cancel()

	// Success case.
	err := retry.Do(ctx, func(ctx context.Context) error {
		return nil
	}, 3)
	fmt.Println(err)

	// Timeout case.
	err = retry.Do(ctx, func(ctx context.Context) error {
		return errors.ErrUnsupported
	}, 3)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, context.DeadlineExceeded))

	// Limit exceeded case.
	err = retry.Do(context.Background(), func(ctx context.Context) error {
		return errors.ErrUnsupported
	}, 1)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, retry.ErrLimitExceeded))

	// Output:
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
