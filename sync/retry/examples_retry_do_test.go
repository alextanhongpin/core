package retry_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Do() {
	err := retry.New(10).Do(context.Background(), func(ctx context.Context) error {
		return nil
	})
	fmt.Println(err)
	// Output:
	// <nil>
}
