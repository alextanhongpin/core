package promise

import (
	"fmt"
	"sync"
)

// Map is a concurrent map of promises keyed by string.
type Map[T any] struct {
	mu sync.RWMutex
	ps map[string]*Promise[T]
}

func NewMap[T any]() *Map[T] {
	return &Map[T]{
		ps: make(map[string]*Promise[T]),
	}
}

func (g *Map[T]) Delete(key string) bool {
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

func (g *Map[T]) LoadOrStore(key string) (*Promise[T], bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.ps[key]
	if ok {
		return p, true
	}

	p = Deferred[T]()
	g.ps[key] = p
	return p, false
}

func (g *Map[T]) Load(key string) (*Promise[T], bool) {
	g.mu.Lock()
	p, ok := g.ps[key]
	g.mu.Unlock()

	return p, ok
}

func (g *Map[T]) LoadMany(keys ...string) map[string]*Promise[T] {
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

func (g *Map[T]) Len() int {
	g.mu.RLock()
	n := len(g.ps)
	g.mu.RUnlock()

	return n
}
