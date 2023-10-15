package retry_test

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetrySkip() {
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
	_, res, err := r.Do(func() (any, error) {
		err := errs[i]
		i++
		return nil, err
	})
	fmt.Println(err)
	fmt.Println(res.Attempts)
	// Output:
	// skipable
	// 1
}
