package throttle

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/evaltest"
)

// TestThrottleDo uses evaltest to run data-driven scenarios for Throttler.Do.
// Each case creates a fresh Throttler with the given limits and performs
// `Calls` consecutive Do invocations. The test asserts the error from the
// final call.
func TestThrottle(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, cfg *Config) (any, error) {
		cfg.BacklogTimeout = 10 * time.Millisecond
		th := New(cfg)

		fn := Func(func(ctx context.Context, req any) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return nil, nil
		}, th)

		var wg sync.WaitGroup
		for i := range 3 {
			wg.Go(func() {
				time.Sleep(time.Duration(i) * 10 * time.Millisecond)
				res, err := fn(ctx, nil)
				evaltest.Log(ctx, evaltest.NewT[any]("fn", nil, res, err))
			})
		}
		wg.Wait()
		return nil, nil
	})
}
