package retry_test

import (
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

	r := retry.New[any](opt)
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}

	start := time.Now()
	v, res, err := r.Do(func() (any, error) {
		return nil, errors.New(http.StatusText(http.StatusInternalServerError))
	})
	elapsed := time.Since(start)

	fmt.Println(v)
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
	// <nil>
	// retry.Result: {Attempts:5 Duration:500ms}
	// retry: max attempts reached - retry 5 times, took 500ms: Internal Server Error
	// true
	// true
}
