// package expire queues deadlines in a given time window and executes the
// handler when the queue size hits the threshold.
package expire

import (
	"context"
	"sync"
	"time"

	"github.com/alextanhongpin/core/types/sliceutil"
	"golang.org/x/exp/event"
)

type Handler func(ctx context.Context) error

func (h Handler) Exec(ctx context.Context) error {
	return h(ctx)
}

type Worker struct {
	count     int           // The current count.
	threshold int           // The maximum count (or unit bytes) within a time interval.
	cond      *sync.Cond    // For conditional wait.
	interval  time.Duration // The time interval.
	times     []time.Time   // The list of times to be executed from first to last.
}

type Option struct {
	Threshold int
	Interval  time.Duration
}

func New(opt Option) *Worker {
	return &Worker{
		interval:  opt.Interval,
		threshold: opt.Threshold,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

// Add adds a new deadline to execute. The next deadline is calculated by
// rounding the deadline to the next interval.
func (w *Worker) Add(deadline time.Time) {
	// Round up the deadline to batch ttls.
	next := deadline.Truncate(w.interval).Add(w.interval)

	c := w.cond
	c.L.Lock()

	w.count++

	if w.isPast(next) {
		c.L.Unlock()

		return
	}

	if w.isCheckpoint() {
		w.count = 0
		w.times = append(w.times, next)
		w.times = sliceutil.Dedup(w.times)
		c.Broadcast()
	}

	c.L.Unlock()
}

// Run executes a background job that handles the execution of the handler when
// the deadline is exceeded.
func (w *Worker) Run(ctx context.Context, h Handler) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer wg.Done()
		defer cancel()

		w.loop(ctx, h)
	}()

	return func() {
		cancel()
		// Always broadcast to unlock sync.Cond and terminate the goroutine.
		w.cond.Broadcast()

		wg.Wait()
	}
}

func (w *Worker) loop(ctx context.Context, h Handler) {
	for {
		c := w.cond
		c.L.Lock()

		// There are two conditions for our sync.Cond to wait:
		// 1. there are no deadline in the queue.
		// 2. the context is not done.
		for len(w.times) == 0 {
			// Before and after waiting, check if it is done.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Wait until there is an expiry deadline.
			c.Wait()

			select {
			case <-ctx.Done():
				c.L.Unlock()

				return
			default:
			}
		}

		// Take the next deadline to wait for.
		next := w.times[0]

		// Remove the deadline.
		w.times = w.times[1:]

		c.L.Unlock()

		// Calculate the sleep duration.
		sleep := next.Sub(time.Now())
		if sleep < 0 {
			continue
		}

		// Sleep until the next deadline.
		<-time.After(sleep)

		// Execute the handler.
		if err := h.Exec(ctx); err != nil {
			event.Log(ctx, err.Error())
		}
	}
}

func (w *Worker) isPast(deadline time.Time) bool {
	if len(w.times) == 0 {
		return false
	}

	return deadline.Before(w.times[0])
}

func (w *Worker) isCheckpoint() bool {
	return w.count >= w.threshold
}
