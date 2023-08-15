package expire

import (
	"context"
	"sync"
	"time"

	"golang.org/x/exp/slices"

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

func (w *Worker) Add(deadline time.Time) {
	// Round up the deadline to batch ttls.
	next := deadline.Truncate(w.interval).Add(w.interval)

	c := w.cond
	c.L.Lock()

	w.count++
	if len(w.times) > 0 && next.Before(w.times[0]) {
		c.L.Unlock()

		return
	}

	if w.count >= w.threshold {
		w.count = 0
		w.times = append(w.times, next)
		// Sort in ascending order.
		slices.SortFunc(w.times, func(a, b time.Time) bool {
			return a.Before(b)
		})
		w.times = slices.Compact(w.times)
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
		w.cond.Broadcast()

		wg.Wait()
	}
}

func (w *Worker) loop(ctx context.Context, h Handler) {
	for {
		c := w.cond
		c.L.Lock()

		for len(w.times) == 0 {
			select {
			case <-ctx.Done():
				return
			default:
			}

			c.Wait()

			select {
			case <-ctx.Done():
				c.L.Unlock()

				return
			default:
			}
		}

		// Shift.
		next := w.times[0]
		w.times = w.times[1:]

		c.L.Unlock()

		// Sleep until the execution.
		sleep := next.Sub(time.Now())
		if sleep < 0 {
			continue
		}

		<-time.After(sleep)

		if err := h.Exec(ctx); err != nil {
			event.Log(ctx, err.Error())
		}
	}
}
