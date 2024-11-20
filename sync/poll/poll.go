package poll

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

var EOQ = errors.New("poll: end of queue")

type Poll struct {
	BatchSize        int
	FailureThreshold int64
	Interval         func(idle int) time.Duration
	Logger           *slog.Logger
	MaxConcurrency   int
}

func (p *Poll) Poll(fn func() error) func() {
	var (
		batchSize        = p.BatchSize
		done             = make(chan struct{})
		failureThreshold = p.FailureThreshold
		idle             = 0
		interval         = p.Interval
		maxConcurrency   = p.MaxConcurrency
	)

	poll := func() error {
		var failures atomic.Int64

		g, ctx := errgroup.WithContext(context.Background())
		g.SetLimit(maxConcurrency)

		p.Logger.Info("poll",
			slog.String("event", "init"),
			slog.Int("batch_size", batchSize),
		)
	loop:
		for i := range batchSize {
			select {
			case <-done:
				p.Logger.Info("poll",
					slog.String("event", "done"),
					slog.Int("i", i))

				break loop
			case <-ctx.Done():
				p.Logger.Info("poll",
					slog.String("event", "ctx.done"),
					slog.Int("i", i))

				break loop
			default:
				g.Go(func() error {
					err := fn()
					if errors.Is(err, EOQ) {
						p.Logger.Info("poll",
							slog.String("event", "eoq"),
							slog.Int("i", i))

						return EOQ
					}
					if err == nil {
						p.Logger.Info("poll",
							slog.String("event", "success"),
							slog.Int("i", i))

						// Decrement for every success after failure.
						if failures.Load() > 0 {
							failures.Add(-1)
						}

						return nil
					}

					// Increment for every unhandled error.
					// Consecutive errors will result in termination.
					failureCount := failures.Add(1)
					if failureCount > failureThreshold {
						p.Logger.Info("poll",
							slog.String("event", "terminate"),
							slog.Int64("count", failureCount),
							slog.String("err", err.Error()),
							slog.Int("i", i))
						return err
					}

					p.Logger.Info("poll",
						slog.String("event", "error"),
						slog.String("err", err.Error()),
						slog.Int("i", i))

					return nil
				})
			}
		}

		return g.Wait()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		sleep := interval(idle)
		p.Logger.Info("batch",
			slog.String("event", "sleep"),
			slog.Duration("sleep", sleep))
		for {
			select {
			case <-done:
				return
			case <-time.After(sleep):
				if err := poll(); err != nil {
					if errors.Is(err, EOQ) {
						idle++

						p.Logger.Info("batch",
							slog.String("event", "idle"),
							slog.Int("idle", idle))

						continue
					}

					p.Logger.Info("batch",
						slog.String("event", "error"),
						slog.String("err", err.Error()))

					// Terminate.
					return
				}

				p.Logger.Info("batch",
					slog.String("event", "success"))

				idle = 0
			}
		}
	}()

	return sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})
}

func ExponentialInterval(idle int) time.Duration {
	idle = min(idle, 6) // Up to 2^6 = 64
	seconds := math.Pow(2, float64(idle))
	return time.Duration(seconds) * time.Second
}
