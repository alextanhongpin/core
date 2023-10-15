package retry_test

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/alextanhongpin/core/sync/retry"
)

func init() {
	retry.Rand = rand.New(rand.NewSource(42))
}

func ExampleRetryShouldHandle() {
	var skipErr = errors.New("skipable")

	var i int
	errs := []error{
		errors.New("random"),
		errors.New("random"),
		skipErr,
	}

	opt := retry.NewOption()
	r := retry.New[any](opt)
	r.ShouldHandle = func(v any, err error) (bool, error) {
		return !errors.Is(err, skipErr), err
	}
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}
	v, res, err := r.Do(func() (any, error) {
		err := errs[i]
		i++
		return nil, err
	})
	fmt.Println(v)
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// retry.Event: {Attempt:1 Delay:130ms Err:random}
	// retry.Event: {Attempt:2 Delay:270ms Err:random}
	// <nil>
	// retry.Result: {Attempts:2 Duration:400ms}
	// skipable
}
