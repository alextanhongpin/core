package expire

import (
	"context"
	"sync"
	"time"

	"golang.org/x/exp/event"
)

type Handler func(ctx context.Context) error

func (h Handler) Exec(ctx context.Context) error {
	return h(ctx)
}

type Worker struct {
	count     int           // The current count.
	every     int           // The maximum count within a time window.
	cond      *sync.Cond    // For conditional wait.
	window    time.Duration // The time window.
	deadlines []time.Time   // The list of deadlines to be executed from first to last.
	last      time.Time     // The last execution time.
}

type Option struct {
	Every  int
	Window time.Duration
}

func New(opt Option) *Worker {
	return &Worker{
		window: opt.Window,
		every:  opt.Every,
		cond:   sync.NewCond(&sync.Mutex{}),
	}
}

func (w *Worker) Add(deadline time.Time) {
	// Round up the deadline to batch ttls.
	next := deadline.Truncate(w.window).Add(w.window)

	c := w.cond
	c.L.Lock()

	// The current batch is already executing.
	// Skip.
	// Alternative is to add to the next window.
	if w.last == next {
		c.L.Unlock()

		return
	}

	// No deadlines yet. Add and broadcast.
	if len(w.deadlines) == 0 {
		w.count++
		w.deadlines = append(w.deadlines, next)
		c.Broadcast()
		c.L.Unlock()

		return
	}

	// Last deadline is within the same window.
	if w.deadlines[len(w.deadlines)-1] == next {
		w.count++
		c.Broadcast()
		c.L.Unlock()

		return
	}

	// Threshold not yet exceeded, push to the next window.
	if w.count < w.every {
		w.count++
		w.deadlines[len(w.deadlines)-1] = next

		c.Broadcast()
		c.L.Unlock()

		return
	}

	// Threshold exceeded. Add as a new time window.
	w.deadlines = append(w.deadlines, next)
	w.count = 1

	c.Broadcast()
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

	cond:
		for {
			select {
			case <-ctx.Done():
				c.L.Unlock()

				return
			default:
				// Wait if there are no deadlines.
				if len(w.deadlines) == 0 {
					c.Wait()
				}

				break cond
			}
		}

		// Shift.
		next := w.deadlines[0]
		w.deadlines = w.deadlines[1:]

		// Ensures that after shifting, no new similar deadline was pushed to the
		// array, which will trigger the execution twice.
		w.last = next

		c.L.Unlock()

		// Sleep until the execution.
		sleep := next.Sub(time.Now())
		<-time.After(sleep)

		if err := h.Exec(ctx); err != nil {
			event.Log(ctx, err.Error())
		}
	}
}
