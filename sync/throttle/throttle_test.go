package throttle

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/evaltest"
)

// TestThrottle schedules calls away from slot-release boundaries in virtual time.
// Results are logged sequentially because evaltest.Log is not concurrency-safe.
func TestThrottle(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, cfg *Config) (any, error) {
		synctest.Test(t, func(t *testing.T) {
			cfg.BacklogTimeout = 10 * time.Millisecond
			th := New(cfg)

			fn := Func(func(ctx context.Context, req any) (any, error) {
				time.Sleep(20 * time.Millisecond)
				return nil, nil
			}, th)

			var wg sync.WaitGroup
			results := make([]error, 3)
			for i := range 3 {
				wg.Go(func() {
					time.Sleep([]time.Duration{0, 5 * time.Millisecond, 25 * time.Millisecond}[i])
					_, results[i] = fn(context.Background(), nil)
				})
			}
			wg.Wait()
			for _, err := range results {
				evaltest.Log(ctx, evaltest.NewT[any, any]("fn", nil, nil, err))
			}
		})
		return nil, nil
	})
}
