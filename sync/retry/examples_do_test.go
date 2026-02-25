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

	collector := &retry.AtomicRetryMetricsCollector{}
	ret := retry.New().WithMetricsCollector(collector)
	// Success case.
	err := ret.Do(ctx, func(ctx context.Context) error {
		return nil
	}, 3)
	fmt.Println(err)

	// Timeout case.
	err = ret.Do(ctx, func(ctx context.Context) error {
		return errors.ErrUnsupported
	}, 3)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, context.DeadlineExceeded))

	// Limit exceeded case.
	err = ret.Do(context.Background(), func(ctx context.Context) error {
		return errors.ErrUnsupported
	}, 1)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errors.ErrUnsupported))
	fmt.Println(errors.Is(err, retry.ErrLimitExceeded))
	fmt.Printf("%#v\n", collector.GetMetrics())

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
	// retry.RetryMetrics{Attempts:5, Successes:2, Failures:5, Throttles:0, LimitExceeded:1}
}
