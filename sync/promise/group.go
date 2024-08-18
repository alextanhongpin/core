package promise

import (
	"errors"
	"fmt"
	"sync"
)

var ErrAborted = errors.New("promise: aborted")

type Group[T any] struct {
	mu sync.Mutex
	ps map[string]*Promise[T]
}

func NewGroup[T any]() *Group[T] {
	return &Group[T]{
		ps: make(map[string]*Promise[T]),
	}
}

func (g *Group[T]) Forget(key string) bool {
	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		delete(g.ps, key)
		g.mu.Unlock()

		// Reject to prevent goroutine leak.
		p.Reject(fmt.Errorf("%w: key replaced", ErrAborted))
		return true
	}

	g.mu.Unlock()
	return false
}

func (g *Group[T]) Do(key string, fn func() (T, error)) (T, error) {
	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		g.mu.Unlock()
		return p.Await()
	}
	p = New(fn)
	g.ps[key] = p
	g.mu.Unlock()

	return p.Await()
}

func (g *Group[T]) LoadOrStore(key string) (*Promise[T], bool) {
	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		g.mu.Unlock()
		return p, true
	}

	p = Deferred[T]()
	g.ps[key] = p
	g.mu.Unlock()
	return p, false
}
