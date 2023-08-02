package background

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

var maxWorkers = runtime.GOMAXPROCS(0)

type task[T any] interface {
	Exec(T)
}

var Stopped = errors.New("background: already stopped")

type Worker[T any] struct {
	sem   *semaphore.Weighted
	wg    sync.WaitGroup
	ch    chan T
	done  chan struct{}
	task  task[T]
	begin sync.Once
	end   sync.Once
	set   sync.Once
}

// New returns a new background manager.
func New[T any](task task[T]) (*Worker[T], func()) {
	w := &Worker[T]{
		sem:  semaphore.NewWeighted(int64(maxWorkers)),
		ch:   make(chan T),
		done: make(chan struct{}),
		task: task,
	}

	return w, w.stop
}

func (w *Worker[T]) SetMaxWorkers(n int) {
	w.set.Do(func() {
		w.sem = semaphore.NewWeighted(int64(n))
	})
}

// Send sends a new message to the channel.
func (w *Worker[T]) Send(t T) error {
	w.init()

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
func (w *Worker[T]) init() {
	w.begin.Do(func() {
		w.wg.Add(1)
		w.set.Do(func() {})

		go func() {
			defer w.wg.Done()

			w.worker()
		}()
	})
}

// stop stops the channel and waits for the channel messages to be flushed.
func (w *Worker[T]) stop() {
	w.end.Do(func() {
		close(w.done)

		w.wg.Wait()
	})
}

// worker listens to the channel for new messages.
func (w *Worker[T]) worker() {
	defer w.flush()

	for {
		select {
		case <-w.done:
			return
		case v := <-w.ch:
			w.exec(v)
		}
	}
}

func (w *Worker[T]) exec(v T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := w.sem.Acquire(ctx, 1); err != nil {
		// Execute the task immediately if we fail to acquire semaphore.
		w.task.Exec(v)

		return
	}

	go func() {
		defer w.sem.Release(1)

		w.task.Exec(v)
	}()
}

func (w *Worker[T]) flush() {
	_ = w.sem.Acquire(context.Background(), int64(maxWorkers))
}
