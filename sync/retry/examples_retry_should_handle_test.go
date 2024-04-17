package retry_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/alextanhongpin/core/sync/retry"
)

func init() {
	retry.Rand = rand.New(rand.NewSource(42))
}

func ExampleRetrySkip() {
	var skipErr = errors.New("skipable")

	var i int
	errs := []error{
		errors.New("random"),
		errors.New("random"),
		skipErr,
	}

	opt := retry.NewOption()
	r := retry.New(opt)
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
	// retry.Event: {Attempt:1 Delay:130ms Err:random}
	// retry.Event: {Attempt:2 Delay:270ms Err:random}
	// retry.Result: retry 2 times, took 400ms
	// <nil>
}
