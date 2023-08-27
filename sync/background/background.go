package background

import (
	"context"
	"runtime"
	"sync"

	"golang.org/x/exp/event"
	"golang.org/x/sync/semaphore"
)

var (
	processedTotal = event.NewCounter("processed_total", &event.MetricOptions{
		Description: "the number of processed async message",
	})

	workersCount = event.NewCounter("workers_count", &event.MetricOptions{
		Description: "the number of goroutines spawn by the workers",
	})
)

type handler[T any] interface {
	Exec(context.Context, T)
}

type Task[T any] func(context.Context, T)

func (h Task[T]) Exec(ctx context.Context, t T) {
	h(ctx, t)
}

type Worker[T any] struct {
	sem     *semaphore.Weighted
	wg      sync.WaitGroup
	handler handler[T]
}

type Option[T any] struct {
	Handler    handler[T]
	MaxWorkers int
}

// New returns a new background manager.
func New[T any](opt Option[T]) (*Worker[T], func()) {
	if opt.MaxWorkers <= 0 {
		opt.MaxWorkers = runtime.GOMAXPROCS(0)
	}

	w := &Worker[T]{
		sem:     semaphore.NewWeighted(int64(opt.MaxWorkers)),
		handler: opt.Handler,
	}

	return w, w.stop
}

// Exec sends a new message to the channel.
func (w *Worker[T]) Exec(ctx context.Context, vs ...T) {
	processedTotal.Record(ctx, int64(len(vs)))

	for _, v := range vs {
		v := v

		w.exec(ctx, v)
	}
}

func (w *Worker[T]) ExecWait(ctx context.Context, vs ...T) {
	processedTotal.Record(ctx, int64(len(vs)))

	var wg sync.WaitGroup
	wg.Add(len(vs))

	for _, v := range vs {
		v := v

		go func(v T) {
			defer wg.Done()

			w.handler.Exec(ctx, v)
		}(v)
	}

	wg.Wait()
}

// ExecWaitN is similar to ExecWait, excepts it limits the running goroutine to
// size n. Executes everything concurrently if the number of messages is less
// than n.
func (w *Worker[T]) ExecWaitN(ctx context.Context, n int, vs ...T) {
	processedTotal.Record(ctx, int64(len(vs)))

	if len(vs) < n {
		w.ExecWait(ctx, vs...)

		return
	}

	sem := semaphore.NewWeighted(int64(n))

	var wg sync.WaitGroup
	wg.Add(len(vs))

	for _, v := range vs {
		v := v

		// If we fail to acquire a semaphore, just run it synchronously.
		if err := sem.Acquire(ctx, 1); err != nil {
			w.handler.Exec(ctx, v)
			continue
		}

		go func(v T) {
			defer wg.Done()
			defer sem.Release(1)

			w.handler.Exec(ctx, v)
		}(v)
	}

	wg.Wait()
}

// stop stops the channel and waits for the channel messages to be flushed.
func (w *Worker[T]) stop() {
	w.wg.Wait()
}

func (w *Worker[T]) exec(ctx context.Context, v T) {
	ctx = context.WithoutCancel(ctx)

	if err := w.sem.Acquire(context.Background(), 1); err != nil {
		event.Error(ctx, "failed to acquire semaphore", err)

		// Execute the handler immediately if we fail to acquire semaphore.
		w.handler.Exec(ctx, v)

		return
	}

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer w.sem.Release(1)

		workersCount.Record(ctx, 1)
		defer workersCount.Record(ctx, -1)

		w.handler.Exec(ctx, v)
	}()
}
