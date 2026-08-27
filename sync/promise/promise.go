package promise

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Status int

const (
	StatusPending Status = iota
	StatusFulfilled
	StatusRejected
)

type Promise[T any] struct {
	cancel func(error)
	done   *atomic.Bool
	once   sync.Once
	fn     func() (T, error)
	ch     chan result[T]
}

type result[T any] struct {
	Data  T
	Error error
}

type handler[T any] = func(ctx context.Context) (T, error)

func New[T any](ctx context.Context, fn handler[T]) *Promise[T] {
	p, ctx := Deferred[T](ctx)
	go p.run(ctx, fn)
	return p
}

func Deferred[T any](ctx context.Context) (*Promise[T], context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	p := &Promise[T]{
		ch:     make(chan result[T], 1),
		done:   new(atomic.Bool),
		cancel: cancel,
	}
	p.fn = sync.OnceValues(func() (T, error) {
		defer p.done.Store(true)
		select {
		case <-ctx.Done():
			var zero T
			return zero, context.Cause(ctx)
		case res := <-p.ch:
			return res.Data, res.Error
		}
	})
	return p, ctx
}

func (p *Promise[T]) run(ctx context.Context, fn handler[T]) {
	p.once.Do(func() {
		res, err := fn(ctx)
		p.ch <- result[T]{Data: res, Error: err}
	})
}

func Resolve[T any](ctx context.Context, v T) *Promise[T] {
	p, _ := Deferred[T](ctx)
	p.Resolve(v)
	return p
}

func Reject[T any](ctx context.Context, err error) *Promise[T] {
	p, _ := Deferred[T](ctx)
	p.Reject(err)
	return p
}

func WithResolvers[T any](ctx context.Context) (p *Promise[T], resolve func(T), reject func(error)) {
	p, _ = Deferred[T](ctx)
	return p, p.Resolve, p.Reject
}

func (p *Promise[T]) Resolve(v T) {
	p.once.Do(func() {
		p.ch <- result[T]{Data: v}
	})
}

func (p *Promise[T]) Reject(err error) {
	p.once.Do(func() {
		p.ch <- result[T]{Error: err}
	})
}

func (p *Promise[T]) Abort(cause error) {
	p.cancel(cause)
}

func (p *Promise[T]) Await() (T, error) {
	return p.fn()
}

func (p *Promise[T]) Status() Status {
	if !p.done.Load() {
		return StatusPending
	}
	_, err := p.Await()
	if err != nil {
		return StatusRejected
	}
	return StatusFulfilled
}

type Result[T any] struct {
	Status Status
	Data   T
	Error  error
}

// Promise.race() settles as soon as the first promise finishes
// (whether it succeeds or fails)
func Race[T any](promises ...*Promise[T]) (T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ch := make(chan result[T], len(promises))
	go func() {
		var wg sync.WaitGroup
		for _, p := range promises {
			wg.Go(func() {
				res, err := p.Await()
				ch <- result[T]{Data: res, Error: err}
			})
		}
		wg.Wait()
		close(ch)
	}()
	defer func() {
		// Flush all
		for range ch {
		}
	}()
	res := <-ch
	return res.Data, res.Error
}

// Promise.any() settles as soon as the first promise succeeds
// (ignoring failures unless they all fail)
func Any[T any](promises ...*Promise[T]) (T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ch := make(chan result[T], len(promises))
	go func() {
		var wg sync.WaitGroup
		for _, p := range promises {
			wg.Go(func() {
				res, err := p.Await()
				ch <- result[T]{Data: res, Error: err}
			})
		}
		wg.Wait()
		close(ch)
	}()
	defer func() {
		// Flush all
		for range ch {
		}
	}()

	var errs []error
	for v := range ch {
		if v.Error != nil {
			errs = append(errs, v.Error)
			continue
		}
		return v.Data, nil
	}

	var zero T
	return zero, errors.Join(errs...)
}

func All[T any](promises ...*Promise[T]) ([]T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ps := AllSettled(promises...)
	res := make([]T, len(ps))
	for i, v := range ps {
		if v.Error != nil {
			return nil, v.Error
		}
		res[i] = v.Data
	}

	return res, nil
}

func AllSettled[T any](promises ...*Promise[T]) []*Result[T] {
	if len(promises) == 0 {
		panic("no promises")
	}
	res := make([]*Result[T], len(promises))
	for i, p := range promises {
		v, err := p.Await()
		if err != nil {
			res[i] = &Result[T]{Error: err, Status: StatusRejected}
		} else {
			res[i] = &Result[T]{Data: v, Status: StatusFulfilled}
		}
	}
	return res
}

type Map[K comparable, V any] = Cache[K, Promise[V]]

func NewMap[K comparable, V any](ctx context.Context) *Map[K, V] {
	return NewCache(func(K) (*Promise[V], error) {
		d, _ := Deferred[V](ctx)
		return d, nil
	})
}
