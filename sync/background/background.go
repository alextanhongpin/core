// Package background implements functions to execute tasks in a separate
// goroutine.
package background

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var ErrTerminated = errors.New("worker: terminated")

// Metrics contains runtime metrics for the worker pool.
type Metrics struct {
	TasksQueued    int64
	TasksProcessed int64
	TasksRejected  int64
	ActiveWorkers  int64
}

// Options configures the worker pool behavior.
type Options struct {
	// WorkerCount is the number of worker goroutines.
	// If 0 or negative, uses runtime.GOMAXPROCS(0).
	WorkerCount int

	// BufferSize is the size of the task channel buffer.
	// If 0, uses unbuffered channel.
	BufferSize int

	// WorkerTimeout is the maximum time a worker can spend processing a task.
	// If 0, no timeout is applied.
	WorkerTimeout time.Duration

	// OnError is called when a worker panics during task processing.
	// The function receives the task and the recovered panic value.
	OnError func(task interface{}, recovered interface{})

	// OnTaskComplete is called after each task is processed.
	// The function receives the task and the processing duration.
	OnTaskComplete func(task interface{}, duration time.Duration)
}

type Worker[T any] struct {
	ch      chan T
	ctx     context.Context
	fn      func(ctx context.Context, v T)
	n       int
	opts    Options
	metrics Metrics
}

// New returns a new background manager.
func New[T any](ctx context.Context, n int, fn func(context.Context, T)) (*Worker[T], func()) {
	opts := Options{
		WorkerCount: n,
		BufferSize:  0,
	}
	return NewWithOptions(ctx, opts, fn)
}

// NewWithOptions returns a new background manager with custom options.
func NewWithOptions[T any](ctx context.Context, opts Options, fn func(context.Context, T)) (*Worker[T], func()) {
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = runtime.GOMAXPROCS(0)
	}

	var ch chan T
	if opts.BufferSize > 0 {
		ch = make(chan T, opts.BufferSize)
	} else {
		ch = make(chan T)
	}

	w := &Worker[T]{
		ch:   ch,
		fn:   fn,
		n:    opts.WorkerCount,
		opts: opts,
	}

	return w, w.init(ctx)
}

// Send sends a new message to the channel.
func (w *Worker[T]) Send(vs ...T) error {
	for _, v := range vs {
		select {
		case <-w.ctx.Done():
			atomic.AddInt64(&w.metrics.TasksRejected, 1)
			return context.Cause(w.ctx)
		case w.ch <- v:
			atomic.AddInt64(&w.metrics.TasksQueued, 1)
		}
	}

	return nil
}

// TrySend attempts to send a message without blocking.
// Returns true if the message was sent, false if the channel is full.
func (w *Worker[T]) TrySend(v T) bool {
	select {
	case <-w.ctx.Done():
		atomic.AddInt64(&w.metrics.TasksRejected, 1)
		return false
	case w.ch <- v:
		atomic.AddInt64(&w.metrics.TasksQueued, 1)
		return true
	default:
		atomic.AddInt64(&w.metrics.TasksRejected, 1)
		return false
	}
}

// Metrics returns a copy of the current metrics.
func (w *Worker[T]) Metrics() Metrics {
	return Metrics{
		TasksQueued:    atomic.LoadInt64(&w.metrics.TasksQueued),
		TasksProcessed: atomic.LoadInt64(&w.metrics.TasksProcessed),
		TasksRejected:  atomic.LoadInt64(&w.metrics.TasksRejected),
		ActiveWorkers:  atomic.LoadInt64(&w.metrics.ActiveWorkers),
	}
}

func (w *Worker[T]) init(ctx context.Context) func() {
	ctx, cancel := context.WithCancelCause(ctx)
	w.ctx = ctx

	var wg sync.WaitGroup
	wg.Add(w.n)

	for range w.n {
		go func() {
			defer wg.Done()
			atomic.AddInt64(&w.metrics.ActiveWorkers, 1)
			defer atomic.AddInt64(&w.metrics.ActiveWorkers, -1)

			for {
				select {
				case <-ctx.Done():
					return
				case v := <-w.ch:
					w.processTask(ctx, v)
				}
			}
		}()
	}

	return func() {
		cancel(ErrTerminated)
		wg.Wait()
	}
}

func (w *Worker[T]) processTask(ctx context.Context, task T) {
	var taskCtx context.Context
	var taskCancel context.CancelFunc

	if w.opts.WorkerTimeout > 0 {
		taskCtx, taskCancel = context.WithTimeout(ctx, w.opts.WorkerTimeout)
		defer taskCancel()
	} else {
		taskCtx = ctx
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&w.metrics.TasksProcessed, 1)

		if w.opts.OnTaskComplete != nil {
			w.opts.OnTaskComplete(task, duration)
		}

		if r := recover(); r != nil {
			if w.opts.OnError != nil {
				w.opts.OnError(task, r)
			}
		}
	}()

	w.fn(taskCtx, task)
}
