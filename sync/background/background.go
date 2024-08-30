// Package background implements functions to execute tasks in a separate
// goroutine.
package background

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

var ErrTerminated = errors.New("worker: terminated")

type Worker[T any] struct {
	ch  chan T
	ctx context.Context
	fn  func(ctx context.Context, v T)
	n   int
}

// New returns a new background manager.
func New[T any](ctx context.Context, n int, fn func(context.Context, T)) (*Worker[T], func()) {
	if n <= 0 {
		n = runtime.GOMAXPROCS(0)
	}

	w := &Worker[T]{
		ch: make(chan T),
		fn: fn,
		n:  n,
	}

	return w, w.init(ctx)
}

// Send sends a new message to the channel.
func (w *Worker[T]) Send(vs ...T) error {
	for _, v := range vs {
		select {
		case <-w.ctx.Done():
			return context.Cause(w.ctx)
		case w.ch <- v:
		}
	}

	return nil
}

func (w *Worker[T]) init(ctx context.Context) func() {
	ctx, cancel := context.WithCancelCause(ctx)
	w.ctx = ctx

	var wg sync.WaitGroup
	wg.Add(w.n)

	for range w.n {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case v := <-w.ch:
					w.fn(ctx, v)
				}
			}
		}()
	}

	return func() {
		cancel(ErrTerminated)
		wg.Wait()
	}
}
