package promise

import (
	"context"
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
	return g.LockWithContext(key, context.Background(), func(ctx context.Context) (T, error) {
		return fn()
	})
}

func (g *Group[T]) LockWithContext(key string, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	if fn == nil {
		var zero T
		return zero, ErrNilFunction
	}

	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		g.mu.Unlock()
		return p.AwaitWithContext(ctx)
	}

	p = NewWithContext(ctx, fn)
	g.ps[key] = p
	g.mu.Unlock()
	defer g.Delete(key)

	return p.AwaitWithContext(ctx)
}

func (g *Group[T]) Do(key string, fn func() (T, error)) (T, error) {
	return g.DoWithContext(key, context.Background(), func(ctx context.Context) (T, error) {
		return fn()
	})
}

func (g *Group[T]) DoWithContext(key string, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	if fn == nil {
		var zero T
		return zero, ErrNilFunction
	}

	g.mu.Lock()
	p, ok := g.ps[key]
	if ok {
		g.mu.Unlock()
		return p.AwaitWithContext(ctx)
	}

	p = NewWithContext(ctx, fn)
	g.ps[key] = p
	g.mu.Unlock()

	return p.AwaitWithContext(ctx)
}

func (g *Group[T]) Delete(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.ps[key]
	if ok {
		delete(g.ps, key)
		// Reject to prevent goroutine leak.
		p.Reject(fmt.Errorf("%w: key deleted", ErrAborted))
	}

	return ok
}

func (g *Group[T]) Load(key string) (*Promise[T], bool) {
	g.mu.RLock()
	p, ok := g.ps[key]
	g.mu.RUnlock()

	return p, ok
}

func (g *Group[T]) LoadMany(keys ...string) map[string]*Promise[T] {
	m := make(map[string]*Promise[T])
	g.mu.RLock()
	for _, k := range keys {
		v, ok := g.ps[k]
		if ok {
			m[k] = v
		}
	}
	g.mu.RUnlock()

	return m
}

func (g *Group[T]) Keys() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	keys := make([]string, 0, len(g.ps))
	for k := range g.ps {
		keys = append(keys, k)
	}
	return keys
}

func (g *Group[T]) Len() int {
	g.mu.RLock()
	n := len(g.ps)
	g.mu.RUnlock()

	return n
}

func (g *Group[T]) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Reject all existing promises to prevent leaks
	for _, p := range g.ps {
		p.Reject(fmt.Errorf("%w: group cleared", ErrAborted))
	}

	g.ps = make(map[string]*Promise[T])
}
