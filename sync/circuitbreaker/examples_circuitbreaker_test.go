package circuitbreaker_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

func ExampleCircuitBreaker() {
	opt := circuitbreaker.NewOption()
	opt.BreakDuration = 100 * time.Millisecond
	opt.SamplingDuration = 1 * time.Second

	cb := circuitbreaker.New[any](opt)
	cb.OnStateChanged = func(from, to circuitbreaker.Status) {
		fmt.Printf("status changed from %s to %s\n", from, to)
	}
	fmt.Println("initial status:", cb.Status())

	// Opens after failure ratio exceeded.
	for i := 0; i <= int(opt.FailureThreshold+1); i++ {
		_, _ = cb.Do(func() (any, error) {
			return nil, errors.New("foo")
		})
	}

	// Break duration.
	time.Sleep(105 * time.Millisecond)

	// Recover.
	for i := 0; i <= int(opt.SuccessThreshold+1); i++ {
		_, _ = cb.Do(func() (any, error) {
			return nil, nil
		})
	}
	// Output:
	// initial status: closed
	// status changed from closed to open
	// status changed from open to half-open
	// status changed from half-open to closed
}
