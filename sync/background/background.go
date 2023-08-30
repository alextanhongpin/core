// Package background implements functions to execute tasks in a separate
// goroutine.
package background

import (
	"context"
	"runtime"
	"sync"

	"golang.org/x/exp/event"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	processedTotal = event.NewCounter("processed_total", &event.MetricOptions{
		Description: "the number of processed async message",
	})

	workersCount = event.NewFloatGauge("workers_count", &event.MetricOptions{
		Description: "the number of background workers running",
	})
)

type Worker[T any] struct {
	sem     *semaphore.Weighted
	wg      sync.WaitGroup
	handler func(ctx context.Context, v T)
}

type Option[T any] struct {
	Handler    func(ctx context.Context, v T)
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
func (w *Worker[T]) Exec(ctx context.Context, v T) {
	processedTotal.Record(ctx, 1)
	w.exec(ctx, v)
}

func (w *Worker[T]) BatchExec(ctx context.Context, vs ...T) {
	processedTotal.Record(ctx, int64(len(vs)))

	for _, v := range vs {
		w.exec(ctx, v)
	}
}

// stop stops the channel and waits for the channel messages to be flushed.
func (w *Worker[T]) stop() {
	w.wg.Wait()
}

func (w *Worker[T]) exec(ctx context.Context, v T) {
	ctx = context.WithoutCancel(ctx)

	if err := w.sem.Acquire(context.Background(), 1); err != nil {
		// Execute the handler immediately if we fail to acquire semaphore.
		w.handler(ctx, v)

		return
	}

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer w.sem.Release(1)

		workersCount.Record(ctx, 1)
		defer workersCount.Record(ctx, -1)

		w.handler(ctx, v)
	}()
}

func BatchExecN[T any](ctx context.Context, h func(ctx context.Context, v T) error, n int, vs ...T) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(n)

	for _, v := range vs {
		v := v
		g.Go(func() error {
			return h(ctx, v)
		})
	}

	return g.Wait()
}
