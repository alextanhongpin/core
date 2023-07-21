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

type Group[T any] struct {
	sem   *semaphore.Weighted
	wg    sync.WaitGroup
	ch    chan T
	done  chan struct{}
	task  task[T]
	begin sync.Once
	end   sync.Once
}

// New returns a new background manager.
func New[T any](task task[T]) (*Group[T], func()) {
	g := &Group[T]{
		sem:  semaphore.NewWeighted(int64(maxWorkers)),
		ch:   make(chan T),
		done: make(chan struct{}),
		task: task,
	}

	return g, g.stop
}

// Send sends a new message to the channel.
func (g *Group[T]) Send(t T) error {
	g.init()

	select {
	case <-g.done:
		return Stopped
	case g.ch <- t:
		// The background worker could be stopped after successfully sending to the
		// channel too.
		select {
		case <-g.done:
			return Stopped
		default:
			return nil
		}
	}
}

// init inits the goroutine that listens for messages from the channel.
func (g *Group[T]) init() {
	g.begin.Do(func() {
		g.wg.Add(1)

		go func() {
			defer g.wg.Done()

			g.worker()
		}()
	})
}

// stop stops the channel and waits for the channel messages to be flushed.
func (g *Group[T]) stop() {
	g.end.Do(func() {
		close(g.done)

		g.wg.Wait()
	})
}

// worker listens to the channel for new messages.
func (g *Group[T]) worker() {
	defer g.flush()

	for {
		select {
		case <-g.done:
			return
		case v := <-g.ch:
			g.exec(v)
		}
	}
}

func (g *Group[T]) exec(v T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := g.sem.Acquire(ctx, 1); err != nil {
		// Execute the task immediately if we fail to acquire semaphore.
		g.task.Exec(v)

		return
	}

	go func() {
		defer g.sem.Release(1)

		g.task.Exec(v)
	}()
}

func (g *Group[T]) flush() {
	_ = g.sem.Acquire(context.Background(), int64(maxWorkers))
}
