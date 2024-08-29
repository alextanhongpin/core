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
	ctx context.Context
	ch  chan T
	fn  func(ctx context.Context, v T)
	sem chan struct{}
}

type Config struct {
	MaxWorkers int
}

func NewConfig() *Config {
	return &Config{
		MaxWorkers: runtime.GOMAXPROCS(0),
	}
}

// New returns a new background manager.
func New[T any](ctx context.Context, fn func(context.Context, T), cfg *Config) (*Worker[T], func()) {
	if cfg == nil {
		cfg = NewConfig()
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = runtime.GOMAXPROCS(0)
	}

	sem := make(chan struct{}, cfg.MaxWorkers)
	for range cfg.MaxWorkers {
		sem <- struct{}{}
	}

	w := &Worker[T]{
		ch:  make(chan T),
		fn:  fn,
		sem: sem,
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
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case v := <-w.ch:
				<-w.sem

				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						w.sem <- struct{}{}
					}()

					w.fn(ctx, v)
				}()
			}
		}
	}()

	return func() {
		cancel(ErrTerminated)
		wg.Wait()
	}
}
