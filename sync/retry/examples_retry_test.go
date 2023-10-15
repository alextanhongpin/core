package retry_test

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry() {
	opt := retry.NewOption()
	opt.Delay = 0
	r := retry.New[any](opt)

	v, res, err := r.Do(func() (any, error) {
		return nil, errors.New("random")
	})
	fmt.Println(v)
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// <nil>
	// retry.Result: {Attempts:10 Duration:0s}
	// retry: max attempts reached - retry 10 times, took 0s: random
}
