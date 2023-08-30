package expire

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/alextanhongpin/core/types/sliceutil"
)

type Queue struct {
	window    time.Duration
	deadlines []time.Time
	next      *time.Time
	cancel    func()
	handler   func()
	mu        sync.Mutex
}

type QueueOption struct {
	Window  time.Duration
	Handler func()
}

func NewQueue(opt QueueOption) *Queue {
	return &Queue{
		window:  opt.Window,
		handler: opt.Handler,
	}
}

func (q *Queue) Add(ctx context.Context, deadline time.Time) {
	// Rounding removes duplicates.
	deadline = round(deadline, q.window)

	q.mu.Lock()
	// Add, sort and remove duplicate deadlines.
	q.deadlines = append(q.deadlines, deadline)
	sort.Slice(q.deadlines, func(i, j int) bool {
		return q.deadlines[i].Before(q.deadlines[j])
	})
	q.deadlines = sliceutil.Dedup(q.deadlines)
	q.mu.Unlock()

	q.enqueue()
}

func (q *Queue) enqueue() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Nothing to enqueue.
	if len(q.deadlines) == 0 {
		return
	}

	next := q.deadlines[0]
	q.deadlines = q.deadlines[1:]

	// There is a pending task to be executed.
	// If the new deadline is the same or if the new deadline is after the next,
	// skip.
	if q.next != nil && (q.next.Equal(next) || q.next.Before(next)) {
		return
	}

	// Cancel pending task before starting a new one.
	if q.cancel != nil {
		q.cancel()
	}

	q.cancel = q.start(next)
}

func (q *Queue) start(deadline time.Time) func() {
	// Assign the next deadline.
	q.next = &deadline

	var wg sync.WaitGroup

	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	// Calculate how long to sleep before the next task.
	sleep := deadline.Sub(time.Now())

	go func() {
		defer wg.Done()

		for {
			select {
			case <-time.After(sleep):
				// Execute and enqueue a new pending task.
				q.handler()
				q.enqueue()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func() {
		cancel()

		wg.Wait()
	}
}

func (q *Queue) Stop() {
	if q.cancel != nil {
		q.cancel()
	}
}

func round(t time.Time, window time.Duration) time.Time {
	return t.Truncate(window).Add(window)
}
