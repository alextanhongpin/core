package singleflight

import (
	"context"
	"sync"
)

// call is an in-flight or completed singleflight remote call
type call[T any] struct {
	val T
	err error
	wg  sync.WaitGroup
}

// Group represents a class of work that may be run in parallel.
type Group[T any] struct {
	mu sync.Mutex
	m  map[string]*call[T]
}

func New[T any]() *Group[T] {
	return &Group[T]{
		m: make(map[string]*call[T]),
	}
}

// Do executes and returns the results of the given function, making sure that only
// one execution is in-flight for a given key at a time. If a duplicate comes in,
// the duplicate caller waits for the original to complete and returns the same results.
func (g *Group[T]) Do(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, bool, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call[T])
	}
	if c, ok := g.m[key]; ok {
		// Wait for the ongoing call to complete.
		g.mu.Unlock()
		c.wg.Wait()

		// Return shared result.
		return c.val, true, c.err
	}

	c := new(call[T])
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	func() {
		defer c.wg.Done()
		c.val, c.err = fn(ctx)

		g.mu.Lock()
		// Remove the call from the map when done.
		delete(g.m, key)
		g.mu.Unlock()
	}()

	// Return the original result.
	return c.val, false, c.err
}
