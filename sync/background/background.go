package background

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

var maxWorkers = runtime.GOMAXPROCS(0)

type Handler[T any] interface {
	Exec(T)
}

type Task[T any] func(T)

func (h Task[T]) Exec(t T) {
	h(t)
}

var Stopped = errors.New("background: already stopped")

type Worker[T any] struct {
	sem     *semaphore.Weighted
	wg      sync.WaitGroup
	ch      chan T
	done    chan struct{}
	handler Handler[T]
	idle    time.Duration
	end     sync.Once
	set     sync.Once
	awake   atomic.Bool
}

type Option[T any] struct {
	IdleTimeout time.Duration
	Handler     Handler[T]
}

// New returns a new background manager.
func New[T any](opt Option[T]) (*Worker[T], func()) {
	if opt.IdleTimeout <= 0 {
		opt.IdleTimeout = 1 * time.Second
	}
	w := &Worker[T]{
		sem:     semaphore.NewWeighted(int64(maxWorkers)),
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
func (w *Worker[T]) Send(t T) error {
	w.start()

	select {
	case <-w.done:
		return Stopped
	case w.ch <- t:
		// The background worker could be stopped after successfully sending to the
		// channel too.
		select {
		case <-w.done:
			return Stopped
		default:
			return nil
		}
	}
}

// init inits the goroutine that listens for messages from the channel.
func (w *Worker[T]) start() {
	if w.awake.Swap(true) {
		return
	}

	w.wg.Add(1)
	w.set.Do(func() {})

	go func() {
		defer w.wg.Done()

		w.worker()
	}()
}

// stop stops the channel and waits for the channel messages to be flushed.
func (w *Worker[T]) stop() {
	w.end.Do(func() {
		close(w.done)
		w.awake.Store(false)

		w.wg.Wait()
	})
}

// worker listens to the channel for new messages.
func (w *Worker[T]) worker() {
	defer w.flush()

	t := time.NewTicker(w.idle)
	defer t.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-t.C:
			w.awake.Store(false)
		case v := <-w.ch:
			w.exec(v)
			t.Reset(w.idle)
		}
	}
}

func (w *Worker[T]) exec(v T) {
	ctx := context.Background()
	if err := w.sem.Acquire(ctx, 1); err != nil {
		// Execute the handler immediately if we fail to acquire semaphore.
		w.handler.Exec(v)

		return
	}

	go func() {
		defer w.sem.Release(1)

		w.handler.Exec(v)
	}()
}

func (w *Worker[T]) flush() {
	_ = w.sem.Acquire(context.Background(), int64(maxWorkers))
}
