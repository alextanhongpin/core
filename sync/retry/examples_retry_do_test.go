package retry_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Do() {
	err := retry.New(10).Do(func() error {
		return nil
	})
	fmt.Println(err)
	// Output:
	// <nil>
}
