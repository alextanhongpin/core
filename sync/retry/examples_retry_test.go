package retry_test

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Try() {
	r := retry.New().WithBackOff(retry.NewConstantBackOff(time.Millisecond))

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	for i, err := range r.Try(ctx, 10) {
		if err != nil {
			fmt.Println(i, err)
			break
		}
	}

	for i, err := range r.Try(context.Background(), 10) {
		if err != nil {
			fmt.Println(i, err)
			break
		}
	}

	// Output:
	// 5 context deadline exceeded
	// 10 retry: limit exceeded
}
