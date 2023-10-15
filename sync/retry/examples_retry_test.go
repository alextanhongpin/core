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

	_, res, err := r.Do(func() (any, error) {
		return nil, errors.New("random")
	})
	fmt.Println(err)
	fmt.Println(res.Attempts)
	fmt.Println(res.Duration)
	// Output:
	// retry: max attempts reached - retry 10 times, took 0s: random
	// 10
	// 0s
}
