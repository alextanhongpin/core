package background

import (
	"context"
	"runtime"
	"sync"

	"golang.org/x/exp/event"
	"golang.org/x/sync/semaphore"
)

var (
	sendCounter = event.NewCounter("background.send", &event.MetricOptions{
		Description: "the number of processed async message",
	})

	sendWaitCounter = event.NewCounter("background.send_wait", &event.MetricOptions{
		Description: "the number of processed sync message",
	})

	goroutineCounter = event.NewCounter("background.goroutine", &event.MetricOptions{
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

// Send sends a new message to the channel.
func (w *Worker[T]) Send(ctx context.Context, vs ...T) {
	sendCounter.Record(ctx, int64(len(vs)))

	for _, v := range vs {
		w.exec(ctx, v)
	}
}

func (w *Worker[T]) SendWait(ctx context.Context, vs ...T) {
	sendWaitCounter.Record(ctx, int64(len(vs)))

	for _, v := range vs {
		w.handler.Exec(ctx, v)
	}
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

		goroutineCounter.Record(ctx, 1)
		defer goroutineCounter.Record(ctx, -1)

		w.handler.Exec(ctx, v)
	}()
}
