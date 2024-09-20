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

	opts := []retry.Option{retry.WithAttempts(10), retry.WithPolicy(noopPolicy)}
	err := retry.DoFunc(func() error {
		return errors.New("random")
	}, opts...)

	fmt.Println(err)

	n, err := retry.DoFunc2(func() (int, error) {
		return 1234, nil
	}, opts...)

	fmt.Println(n, err)
	// Output:
	// retry: aborted: retry: throttled
	// 1234 <nil>
}
