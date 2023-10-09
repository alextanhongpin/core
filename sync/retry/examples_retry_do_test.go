package retry_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetryDo() {
	ctx := context.Background()
	backoffs := retry.Backoffs{0, 0, 0, 0, 0}
	v, err, res := retry.Do(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	}, backoffs)
	fmt.Println(v)
	fmt.Println(err)
	fmt.Println(res.Retry)
	fmt.Println(res.Skip)
	// Output:
	// 42
	// <nil>
	// 0
	// false
}
