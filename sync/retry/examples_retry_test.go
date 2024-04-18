package retry_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func init() {
	rand.Seed(42)
}

func ExampleRetry() {
	r := retry.New(0)
	r.Now = func() time.Time { return time.Time{} }

	ctx := context.Background()
	res, err := r.Do(ctx, func(ctx context.Context) error {
		return errors.New("random")
	})
	fmt.Printf("retry.Result: %+v\n", res)
	fmt.Println(err)
	// Output:
	// retry.Result: &{Retries:[0001-01-01 00:00:00 +0000 UTC]}
	// retry: too many attempts
	// random
}
