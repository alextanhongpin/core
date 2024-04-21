package circuitbreaker_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

func ExampleCircuitBreaker() {
	opt := circuitbreaker.NewOption()
	opt.BreakDuration = 100 * time.Millisecond
	opt.SamplingDuration = 1 * time.Second

	cb := circuitbreaker.New(opt)
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		fmt.Printf("status changed from %s to %s\n", from, to)
	}

	key := "key"
	fmt.Println("initial status:")
	fmt.Println(cb.Status(ctx, key))

	// Opens after failure ratio exceeded.
	for i := 0; i <= int(opt.FailureThreshold+1); i++ {
		_ = cb.Do(ctx, key, func() error {
			return errors.New("foo")
		})
	}

	// Break duration.
	time.Sleep(105 * time.Millisecond)

	// Recover.
	for i := 0; i <= int(opt.SuccessThreshold+1); i++ {
		_ = cb.Do(ctx, key, func() error {
			return nil
		})
	}
	// Output:
	// initial status:
	// closed <nil>
	// status changed from closed to open
	// status changed from open to half-open
	// status changed from half-open to closed
}
