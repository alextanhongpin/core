package background

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/sync/semaphore"
)

var (
	sendCounter    = event.NewCounter("background:send", nil)
	goroutineGauge = event.NewFloatGauge("background:goroutine", nil)
)

type handler[T any] interface {
	Exec(context.Context, T)
}

type Task[T any] func(context.Context, T)

func (h Task[T]) Exec(ctx context.Context, t T) {
	h(ctx, t)
}

var Stopped = errors.New("background: already stopped")

type Worker[T any] struct {
	sem     *semaphore.Weighted
	wg      sync.WaitGroup
	ch      chan T
	done    chan struct{}
	handler handler[T]
	idle    time.Duration
	end     sync.Once
	set     sync.Once
	awake   atomic.Bool
	workers int
}

type Option[T any] struct {
	IdleTimeout time.Duration
	Handler     handler[T]
}

// New returns a new background manager.
func New[T any](opt Option[T]) (*Worker[T], func()) {
	if opt.IdleTimeout <= 0 {
		opt.IdleTimeout = 1 * time.Second
	}
	workers := runtime.GOMAXPROCS(0)
	w := &Worker[T]{
		workers: workers,
		sem:     semaphore.NewWeighted(int64(workers)),
		ch:      make(chan T),
		done:    make(chan struct{}),
		handler: opt.Handler,
		idle:    opt.IdleTimeout,
	}

	return w, w.stop
}

func (w *Worker[T]) SetMaxWorkers(n int) {
	w.set.Do(func() {
		w.sem = semaphore.NewWeighted(int64(n))
	})
}

func (w *Worker[T]) IsIdle() bool {
	return !w.awake.Load()
}

// Send sends a new message to the channel.
func (w *Worker[T]) Send(ctx context.Context, t T) error {
	w.start(ctx)

	select {
	case <-w.done:
		return Stopped
	case w.ch <- t:
		// The background loop could be stopped after successfully sending to the
		// channel too.
		select {
		case <-w.done:
			return Stopped
		default:
			sendCounter.Record(ctx, 1)
			return nil
		}
	}
}

// init inits the goroutine that listens for messages from the channel.
func (w *Worker[T]) start(ctx context.Context) {
	// Swap returns the old bool value.
	if isAwake := w.awake.Swap(true); isAwake {
		return
	}

	w.wg.Add(1)
	w.set.Do(func() {})

	go func() {
		defer w.wg.Done()

		w.loop(ctx)
	}()
}

// stop stops the channel and waits for the channel messages to be flushed.
func (w *Worker[T]) stop() {
	w.end.Do(func() {
		close(w.done)

		w.wg.Wait()
	})
}

// loop listens to the channel for new messages.
func (w *Worker[T]) loop(ctx context.Context) {
	defer w.flush()

	t := time.NewTicker(w.idle)
	defer t.Stop()

	defer w.awake.Store(false)

	for {
		select {
		case <-w.done:
			return
		case <-t.C:
			return
		case v := <-w.ch:
			w.exec(ctx, v)
			t.Reset(w.idle)
		}
	}
}

func (w *Worker[T]) exec(ctx context.Context, v T) {
	ctx = context.WithoutCancel(ctx)
	if err := w.sem.Acquire(context.Background(), 1); err != nil {
		// Execute the handler immediately if we fail to acquire semaphore.
		w.handler.Exec(ctx, v)

		return
	}

	go func() {
		defer w.sem.Release(1)
		goroutineGauge.Record(ctx, 1)
		defer goroutineGauge.Record(ctx, -1)

		w.handler.Exec(ctx, v)
	}()
}

func (w *Worker[T]) flush() {
	n := int64(w.workers)
	ctx := context.Background()

	_ = w.sem.Acquire(ctx, n)
	w.sem.Release(n)
}
