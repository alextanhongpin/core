package promise

import (
	"context"
	"sync"
)

type Pool[T any] struct {
	wg       sync.WaitGroup
	mu       sync.Mutex
	limit    int
	promises Promises[T]
	sem      chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewPool[T any](limit int) *Pool[T] {
	return NewPoolWithContext[T](context.Background(), limit)
}

func NewPoolWithContext[T any](ctx context.Context, limit int) *Pool[T] {
	if limit <= 0 {
		limit = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	sem := make(chan struct{}, limit)
	for range limit {
		sem <- struct{}{}
	}

	return &Pool[T]{
		limit:  limit,
		sem:    sem,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Pool[T]) Do(fn func() (T, error)) error {
	return p.DoWithContext(p.ctx, func(ctx context.Context) (T, error) {
		return fn()
	})
}

func (p *Pool[T]) DoWithContext(ctx context.Context, fn func(context.Context) (T, error)) error {
	if fn == nil {
		return ErrNilFunction
	}

	select {
	case <-p.ctx.Done():
		return ErrCanceled
	case <-ctx.Done():
		return ErrCanceled
	case <-p.sem:
		// Got semaphore, proceed
	}

	p.wg.Add(1)

	pn := NewWithContext(ctx, fn)

	p.mu.Lock()
	p.promises = append(p.promises, pn)
	p.mu.Unlock()

	go func() {
		defer func() {
			p.sem <- struct{}{}
			p.wg.Done()
		}()

		pn.Await()
	}()

	return nil
}

func (p *Pool[T]) All() ([]T, error) {
	p.wg.Wait()

	p.mu.Lock()
	promises := make(Promises[T], len(p.promises))
	copy(promises, p.promises)
	p.mu.Unlock()

	return promises.All()
}

func (p *Pool[T]) AllSettled() []Result[T] {
	p.wg.Wait()

	p.mu.Lock()
	promises := make(Promises[T], len(p.promises))
	copy(promises, p.promises)
	p.mu.Unlock()

	return promises.AllSettled()
}

func (p *Pool[T]) Cancel() {
	p.cancel()
}

func (p *Pool[T]) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.promises)
}

func (p *Pool[T]) Cap() int {
	return p.limit
}
