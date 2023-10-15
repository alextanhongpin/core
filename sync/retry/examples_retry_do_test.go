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
	fmt.Println(err)
	fmt.Println(res.Attempts)
	// Output:
	// 42
	// <nil>
	// 0
}
