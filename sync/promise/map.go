package promise

import (
	"context"
	"fmt"
	"sync"
)

// Map is a concurrent map of promises keyed by string.
// It provides similar functionality to Group but with a cleaner API.
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
		p.Reject(fmt.Errorf("%w: key deleted", ErrAborted))
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

func (g *Map[T]) LoadOrStoreWithContext(key string, ctx context.Context) (*Promise[T], bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.ps[key]
	if ok {
		return p, true
	}

	p = DeferredWithContext[T](ctx)
	g.ps[key] = p
	return p, false
}

func (g *Map[T]) Load(key string) (*Promise[T], bool) {
	g.mu.RLock()
	p, ok := g.ps[key]
	g.mu.RUnlock()

	return p, ok
}

func (g *Map[T]) LoadMany(keys ...string) map[string]*Promise[T] {
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

func (g *Map[T]) Store(key string, promise *Promise[T]) {
	if promise == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// If there's an existing promise, reject it to prevent leaks
	if existing, ok := g.ps[key]; ok {
		existing.Reject(fmt.Errorf("%w: key replaced", ErrAborted))
	}

	g.ps[key] = promise
}

func (g *Map[T]) Keys() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	keys := make([]string, 0, len(g.ps))
	for k := range g.ps {
		keys = append(keys, k)
	}
	return keys
}

func (g *Map[T]) Len() int {
	g.mu.RLock()
	n := len(g.ps)
	g.mu.RUnlock()

	return n
}

func (g *Map[T]) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Reject all existing promises to prevent leaks
	for _, p := range g.ps {
		p.Reject(fmt.Errorf("%w: map cleared", ErrAborted))
	}

	g.ps = make(map[string]*Promise[T])
}

// Do executes the function and stores the resulting promise.
// Similar to Group.Do but with a cleaner interface.
func (g *Map[T]) Do(key string, fn func() (T, error)) (T, error) {
	return g.DoWithContext(key, context.Background(), func(ctx context.Context) (T, error) {
		return fn()
	})
}

// DoWithContext executes the function with context and stores the resulting promise.
func (g *Map[T]) DoWithContext(key string, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
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

// Lock is like Do, but it removes the promise from the map after completion.
// This allows the promise to be garbage collected and mimics singleflight behavior.
func (g *Map[T]) Lock(key string, fn func() (T, error)) (T, error) {
	return g.LockWithContext(key, context.Background(), func(ctx context.Context) (T, error) {
		return fn()
	})
}

// LockWithContext is like DoWithContext, but removes the promise after completion.
func (g *Map[T]) LockWithContext(key string, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
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
