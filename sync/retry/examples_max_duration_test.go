package retry_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleMaxDuration() {
	opt := &retry.Option{
		BackoffType:      retry.BackoffTypeConstant,
		Delay:            100 * time.Millisecond,
		MaxDelay:         time.Second,
		MaxRetryAttempts: 10,
		MaxDuration:      500 * time.Millisecond,
		UseJitter:        false,
	}

	r := retry.New(opt)
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}

	start := time.Now()
	ctx := context.Background()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		return errors.New(http.StatusText(http.StatusInternalServerError))
	})
	elapsed := time.Since(start)

	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	fmt.Println(elapsed > 500*time.Millisecond)
	fmt.Println(elapsed < 510*time.Millisecond)

	// Output:
	// retry.Event: {Attempt:1 Delay:100ms Err:Internal Server Error}
	// retry.Event: {Attempt:2 Delay:100ms Err:Internal Server Error}
	// retry.Event: {Attempt:3 Delay:100ms Err:Internal Server Error}
	// retry.Event: {Attempt:4 Delay:100ms Err:Internal Server Error}
	// retry.Event: {Attempt:5 Delay:100ms Err:Internal Server Error}
	// retry.Result: retry 5 times, took 500ms
	// retry: wait timeout exceeded
	// true
	// true
}
