package promise

import (
	"errors"
	"fmt"
	"sync"
)

var ErrAborted = errors.New("promise: aborted")

type Group[T any] struct {
	mu sync.RWMutex
	ps map[string]*Promise[T]
}

func NewGroup[T any]() *Group[T] {
	return &Group[T]{
		ps: make(map[string]*Promise[T]),
	}
}

// Lock is like Do, but it removes the promise from the group after the
// promise is resolved or rejected.
// This allows the promise to be garbage collected.
// Mimics singleflight behaviour, unless the key is
// already set and the promise is fulfilled or rejected,
// then it behaves like Do.
func (g *Group[T]) Lock(key string, fn func() (T, error)) (T, error) {
	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		g.mu.Unlock()

		return p.Await()
	}

	p = New(fn)
	g.ps[key] = p
	g.mu.Unlock()
	defer g.Delete(key)

	return p.Await()
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

func (g *Group[T]) Delete(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.ps[key]
	if ok {
		delete(g.ps, key)
		// Reject to prevent goroutine leak.
		p.Reject(fmt.Errorf("%w: key replaced", ErrAborted))
	}

	return ok
}

func (g *Group[T]) Load(key string) (*Promise[T], bool) {
	g.mu.Lock()
	p, ok := g.ps[key]
	g.mu.Unlock()

	return p, ok
}

func (g *Group[T]) LoadMany(keys ...string) map[string]*Promise[T] {
	m := make(map[string]*Promise[T])
	g.mu.Lock()
	for _, k := range keys {
		v, ok := g.ps[k]
		if ok {
			m[k] = v
		}
	}
	g.mu.Unlock()

	return m
}

func (g *Group[T]) Len() int {
	g.mu.RLock()
	n := len(g.ps)
	g.mu.RUnlock()

	return n
}
