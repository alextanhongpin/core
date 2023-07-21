package backpressure

import (
	"context"
	"errors"
	"sync"
)

var Closed = errors.New("backpressure: closed")

type Guard struct {
	ch   chan struct{}
	done chan struct{}
	end  sync.Once
}

func New(cap int) *Guard {
	if cap < 1 {
		panic("backpressure: cap must be at least 1")
	}

	return &Guard{
		ch:   make(chan struct{}, cap),
		done: make(chan struct{}),
	}
}

func (g *Guard) Lock(ctx context.Context) error {
	select {
	case <-g.done:
		return Closed
	case <-ctx.Done():
		return context.Cause(ctx)
	case g.ch <- struct{}{}:
		return nil
	}
}

func (g *Guard) Unlock() {
	if len(g.ch) > 0 {
		<-g.ch
	}
}

func (g *Guard) Flush() {
	g.end.Do(func() {
		close(g.done)

		for len(g.ch) > 0 {
			<-g.ch
		}
	})
}
