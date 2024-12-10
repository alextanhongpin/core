package singleflight

import (
	"context"
	"sync"
)

type Group[T any] struct {
	mu    sync.Mutex
	tasks map[string]*task[T]
}

func New[T any]() *Group[T] {
	return &Group[T]{
		tasks: make(map[string]*task[T]),
	}
}

func (g *Group[T]) Do(ctx context.Context, key string, fn func(ctx context.Context) (T, error)) (T, bool, error) {
	g.mu.Lock()
	t, ok := g.tasks[key]
	if ok {
		g.mu.Unlock()
		data, err := t.Unwrap()
		return data, err == nil, err
	}

	t = newTask[T]()
	g.tasks[key] = t
	g.mu.Unlock()

	go func() {
		defer t.wg.Done()

		data, err := fn(ctx)
		t.Data = data
		t.Err = err

		g.mu.Lock()
		delete(g.tasks, key)
		g.mu.Unlock()
	}()

	data, err := t.Unwrap()
	return data, false, err
}

type task[T any] struct {
	wg   *sync.WaitGroup
	Data T
	Err  error
}

func newTask[T any]() *task[T] {
	var wg sync.WaitGroup
	wg.Add(1)

	return &task[T]{
		wg: &wg,
	}
}

func (t *task[T]) Unwrap() (T, error) {
	t.wg.Wait()

	return t.Data, t.Err
}
