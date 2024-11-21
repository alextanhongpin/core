package poll

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"runtime"
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

func New() *Poll {
	return &Poll{
		BatchSize:        1_000,
		FailureThreshold: 25,
		Interval:         ExponentialInterval,
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxConcurrency:   MaxConcurrency(),
	}
}

func (p *Poll) Poll(fn func(context.Context) error) func() {
	var (
		batchSize        = p.BatchSize
		done             = make(chan struct{})
		failureThreshold = p.FailureThreshold
		idle             = 0
		interval         = p.Interval
		logger           = p.Logger
		maxConcurrency   = p.MaxConcurrency
	)

	poll := func(ctx context.Context) error {
		if errors.Is(fn(ctx), EOQ) {
			return EOQ
		}

		var failures atomic.Int64

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(maxConcurrency)

		logger.Info("batch",
			slog.String("event", "init"),
			slog.Int("batch_size", batchSize),
		)
	loop:
		for i := range batchSize {
			select {
			case <-done:
				logger.Info("batch",
					slog.String("event", "done"),
					slog.Int("i", i))

				break loop
			case <-ctx.Done():
				logger.Info("batch",
					slog.String("event", "ctx.done"),
					slog.Int("i", i))

				break loop
			default:
				g.Go(func() error {
					err := fn(ctx)
					if errors.Is(err, EOQ) {
						logger.Info("batch",
							slog.String("event", "eoq"),
							slog.Int("i", i))

						return EOQ
					}
					if err == nil {
						logger.Info("batch",
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
						logger.Info("batch",
							slog.String("event", "terminate"),
							slog.Int64("count", failureCount),
							slog.String("err", err.Error()),
							slog.Int("i", i))
						return err
					}

					logger.Info("batch",
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
		logger.Info("poll",
			slog.String("event", "sleep"),
			slog.Duration("sleep", sleep))
		for {
			select {
			case <-done:
				return
			case <-time.After(sleep):
				if err := poll(context.Background()); err != nil {
					if errors.Is(err, EOQ) {
						idle++

						logger.Info("poll",
							slog.String("event", "idle"),
							slog.Int("idle", idle))

						continue
					}

					logger.Info("poll",
						slog.String("event", "error"),
						slog.String("err", err.Error()))

					// Terminate.
					return
				}

				logger.Info("poll",
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

// ExponentialInterval returns the duration to sleep before the next poll.
// Idle will be zero if there are items in the queue. Otherwise, it will
// increment.
func ExponentialInterval(idle int) time.Duration {
	idle = min(idle, 6) // Up to 2^6 = 64
	seconds := math.Pow(2, float64(idle))
	return time.Duration(seconds) * time.Second
}

func MaxConcurrency() int {
	return min(runtime.GOMAXPROCS(0), runtime.NumCPU())
}
