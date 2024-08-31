// package expire optimally wait for the next deadline
// when there are at least n number of items to expire.
package expire

import (
	"context"
	"sync"
	"time"

	"github.com/alextanhongpin/core/types/sliceutil"
	"golang.org/x/exp/event"
)

var (
	keysTotal = event.NewCounter("keys_total", &event.MetricOptions{
		Description: "the number of keys added",
	})

	requestsTotal = event.NewCounter("requests_total", &event.MetricOptions{
		Description: "the number times the handler executes",
	})

	failuresTotal = event.NewCounter("failures_total", &event.MetricOptions{
		Description: "the number times the handler fails",
	})
)

type Worker struct {
	count     int           // The current count.
	threshold int           // The maximum count (or unit bytes) within a time interval.
	cond      *sync.Cond    // For conditional wait.
	interval  time.Duration // The time interval.
	expiry    time.Duration
	times     []time.Time // The list of times to be executed from first to last.
	last      time.Time   // The last execution time.
}

type Option struct {
	Threshold int           // How many expired keys to keep before cleaning.
	Expiry    time.Duration // How long before a key expires (it must be the same for all).
	Interval  time.Duration // The time window to accumulate the keys.
}

func New(opt Option) *Worker {
	return &Worker{
		interval:  opt.Interval,
		threshold: opt.Threshold,
		expiry:    opt.Expiry,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

// Add adds n number of keys to expire.
func (w *Worker) Add(ctx context.Context, n int) {
	// Round up the deadline to batch ttls.
	next := time.Now().Add(w.expiry).Truncate(w.interval).Add(w.interval)

	c := w.cond
	c.L.Lock()
	defer c.L.Unlock()

	if w.isPast(next) {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	w.count += n
	keysTotal.Record(ctx, int64(n))

	if !w.isCheckpoint() {
		return
	}

	w.count = 0
	w.times = append(w.times, next)
	w.times = sliceutil.Dedup(w.times)
	c.Broadcast()
}

// HandleFunc executes a background job that handles the execution of the handler when
// the deadline is exceeded.
func (w *Worker) HandleFunc(ctx context.Context, h func(ctx context.Context) error) func() {
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

func (w *Worker) loop(ctx context.Context, h func(ctx context.Context) error) {
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
				c.L.Unlock()

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

		select {
		case <-ctx.Done():
			c.L.Unlock()
			return
		default:
		}

		// Take the next deadline to wait for.
		next := w.times[0]
		w.last = next

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
		requestsTotal.Record(ctx, 1)
		if err := h(ctx); err != nil {
			failuresTotal.Record(ctx, 1)
			event.Error(ctx, "failed to execute handler", err)
		}
	}
}

func (w *Worker) isPast(deadline time.Time) bool {
	return deadline == w.last
}

func (w *Worker) isCheckpoint() bool {
	return w.count >= w.threshold
}
