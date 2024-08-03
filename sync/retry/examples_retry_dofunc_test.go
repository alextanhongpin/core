package retry_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_DoFunc() {
	noopPolicy := func(i int) time.Duration {
		return 0
	}

	err := retry.DoFunc(10, noopPolicy, func() error {
		return errors.New("random")
	})

	fmt.Println(err)

	n, err := retry.DoFunc2(10, noopPolicy, func() (int, error) {
		return 1234, nil
	})

	fmt.Println(n, err)
	// Output:
	// retry: limit exceeded
	// random
	// 1234 <nil>
}
