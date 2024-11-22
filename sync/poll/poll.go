package poll

import (
	"context"
	"errors"
	"math"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var EOQ = errors.New("batch: end of queue")

type Poll struct {
	BatchSize        int
	FailureThreshold int
	Interval         func(idle int) time.Duration
	MaxConcurrency   int
}

func New() *Poll {
	return &Poll{
		BatchSize:        1_000,
		FailureThreshold: 25,
		Interval:         ExponentialInterval,
		MaxConcurrency:   MaxConcurrency(),
	}
}

func (p *Poll) Poll(fn func(context.Context) error) (<-chan Event, func()) {
	var (
		batchSize        = p.BatchSize
		ch               = make(chan Event)
		done             = make(chan struct{})
		failureThreshold = p.FailureThreshold
		interval         = p.Interval
		maxConcurrency   = p.MaxConcurrency
	)

	batch := func(ctx context.Context) (err error) {
		limiter := NewLimiter(failureThreshold)
		work := func() error {
			err := limiter.Do(func() error {
				return fn(ctx)
			})
			if errors.Is(err, EOQ) || errors.Is(err, ErrLimitExceeded) {
				return err
			}

			return nil
		}

		defer func(start time.Time) {
			ch <- Event{
				Name: "batch",
				Data: map[string]any{
					"success":  limiter.SuccessCount(),
					"failures": limiter.FailureCount(),
					"total":    limiter.SuccessCount() + limiter.FailureCount(),
					"took":     time.Since(start).Seconds(),
				},
				Err: err,
			}
		}(time.Now())

		if err := work(); err != nil {
			return err
		}

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(maxConcurrency)

	loop:
		for range batchSize - 1 {
			select {
			case <-done:
				break loop
			case <-ctx.Done():
				break loop
			default:
				g.Go(work)
			}
		}

		return g.Wait()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		var idle int

		for {
			sleep := interval(idle)
			select {
			case <-done:
				return
			case ch <- Event{
				Name: "poll",
				Data: map[string]any{
					"sleep": sleep.Seconds(),
					"idle":  idle,
				},
			}:
			}

			select {
			case <-done:
				return
			case <-time.After(sleep):
				if err := batch(context.Background()); err != nil {
					if errors.Is(err, EOQ) {
						idle++

						continue
					}

					return
				}

				idle = 0
			}
		}
	}()

	return ch, sync.OnceFunc(func() {
		close(done)
		wg.Wait()
		close(ch)
	})
}

// ExponentialInterval returns the duration to sleep before the next batch.
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

type Event struct {
	Name string
	Data map[string]any
	Err  error
}
