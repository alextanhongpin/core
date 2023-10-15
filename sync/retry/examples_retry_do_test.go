package retry_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetryDo() {
	r := retry.New[int](nil)
	v, res, err := r.Do(func() (int, error) {
		return 42, nil
	})
	fmt.Println(v)
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// 42
	// retry.Result: {Attempts:0 Duration:0s}
	// <nil>
}
