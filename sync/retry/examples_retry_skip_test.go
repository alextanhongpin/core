package retry_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func init() {
	rand.Seed(42)
}

func ExampleRetrySkip() {
	var skipErr = errors.New("skipable")

	var i int
	errs := []error{
		errors.New("random"),
		errors.New("random"),
		skipErr,
	}

	r := retry.New()
	r.Now = func() time.Time { return time.Time{} }
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}
	ctx := context.Background()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		err := errs[i]
		i++
		if errors.Is(err, skipErr) {
			return nil
		}
		return err
	})
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:1 Delay:51.072305ms Err:random}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:2 Delay:141.734987ms Err:random}
	// retry.Result: &{Retries:[0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC]}
	// <nil>
}
