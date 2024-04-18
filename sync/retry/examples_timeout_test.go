package retry_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func init() {
	rand.Seed(42)
}

func ExampleTimeout() {
	r := retry.New(
		10*time.Millisecond,
		10*time.Millisecond,
		10*time.Millisecond,
		10*time.Millisecond,
	)
	r.JitterFunc = noJitter
	r.Now = func() time.Time {
		return time.Time{}
	}
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 30*time.Millisecond, errors.New("wait timeout exceeded"))
	defer cancel()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		return errors.New(http.StatusText(http.StatusInternalServerError))
	})
	fmt.Printf("retry.Result: %#v\n", res)
	fmt.Println(err)

	// Output:
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:1 Delay:10ms Err:Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:2 Delay:10ms Err:Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:3 Delay:10ms Err:Internal Server Error}
	// retry.Result: &retry.Result{Retries:[]time.Time{time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)}}
	// wait timeout exceeded
}

func noJitter(d time.Duration) time.Duration {
	return d
}
