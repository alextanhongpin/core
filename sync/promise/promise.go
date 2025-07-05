package promise

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrTimeout       = errors.New("promise: timeout")
	ErrCanceled      = errors.New("promise: canceled")
	ErrNilFunction   = errors.New("promise: nil function")
	ErrEmptyPromises = errors.New("promise: empty promises")
)

type Promise[T any] struct {
	wg       sync.WaitGroup
	once     sync.Once
	data     T
	err      error
	ctx      context.Context
	cancel   context.CancelFunc
	resolved atomic.Bool
}

func Deferred[T any]() *Promise[T] {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Promise[T]{
		ctx:    ctx,
		cancel: cancel,
	}
	p.wg.Add(1)
	return p
}

func DeferredWithContext[T any](ctx context.Context) *Promise[T] {
	ctx, cancel := context.WithCancel(ctx)
	p := &Promise[T]{
		ctx:    ctx,
		cancel: cancel,
	}
	p.wg.Add(1)
	return p
}

func Resolve[T any](v T) *Promise[T] {
	return Deferred[T]().Resolve(v)
}

func Reject[T any](err error) *Promise[T] {
	return Deferred[T]().Reject(err)
}

func New[T any](fn func() (T, error)) *Promise[T] {
	if fn == nil {
		return Reject[T](ErrNilFunction)
	}

	p := Deferred[T]()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					p.Reject(err)
				} else {
					p.Reject(errors.New("promise: panic occurred"))
				}
			}
		}()

		select {
		case <-p.ctx.Done():
			p.Reject(ErrCanceled)
			return
		default:
		}

		data, err := fn()
		p.once.Do(func() {
			if err != nil {
				p.err = err
			} else {
				p.data = data
			}
			p.resolved.Store(true)
			p.wg.Done()
		})
	}()

	return p
}

func NewWithContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Promise[T] {
	if fn == nil {
		return Reject[T](ErrNilFunction)
	}

	p := DeferredWithContext[T](ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					p.Reject(err)
				} else {
					p.Reject(errors.New("promise: panic occurred"))
				}
			}
		}()

		select {
		case <-p.ctx.Done():
			p.Reject(ErrCanceled)
			return
		default:
		}

		data, err := fn(p.ctx)
		p.once.Do(func() {
			if err != nil {
				p.err = err
			} else {
				p.data = data
			}
			p.resolved.Store(true)
			p.wg.Done()
		})
	}()

	return p
}

func (p *Promise[T]) Resolve(v T) *Promise[T] {
	p.once.Do(func() {
		p.data = v
		p.resolved.Store(true)
		p.wg.Done()
	})

	return p
}

func (p *Promise[T]) Reject(err error) *Promise[T] {
	p.once.Do(func() {
		p.err = err
		p.resolved.Store(true)
		p.wg.Done()
	})

	return p
}

func (p *Promise[T]) Cancel() {
	p.cancel()
	p.Reject(ErrCanceled)
}

func (p *Promise[T]) Await() (T, error) {
	p.wg.Wait()
	return p.data, p.err
}

func (p *Promise[T]) AwaitWithTimeout(timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.AwaitWithContext(ctx)
}

func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return p.data, p.err
	case <-ctx.Done():
		var zero T
		return zero, ErrTimeout
	}
}

func (p *Promise[T]) IsPending() bool {
	return !p.resolved.Load()
}

func (p *Promise[T]) IsResolved() bool {
	return p.resolved.Load() && p.err == nil
}

func (p *Promise[T]) IsRejected() bool {
	return p.resolved.Load() && p.err != nil
}

type Promises[T any] []*Promise[T]

func (promises Promises[T]) All() ([]T, error) {
	if len(promises) == 0 {
		return []T{}, nil
	}

	res := make([]T, len(promises))

	for i, p := range promises {
		v, err := p.Await()
		if err != nil {
			return nil, err
		}
		res[i] = v
	}

	return res, nil
}

func (promises Promises[T]) AllWithTimeout(timeout time.Duration) ([]T, error) {
	if len(promises) == 0 {
		return []T{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return promises.AllWithContext(ctx)
}

func (promises Promises[T]) AllWithContext(ctx context.Context) ([]T, error) {
	if len(promises) == 0 {
		return []T{}, nil
	}

	res := make([]T, len(promises))

	for i, p := range promises {
		v, err := p.AwaitWithContext(ctx)
		if err != nil {
			return nil, err
		}
		res[i] = v
	}

	return res, nil
}

func (promises Promises[T]) AllSettled() []Result[T] {
	if len(promises) == 0 {
		return []Result[T]{}
	}

	res := make([]Result[T], len(promises))

	for i, p := range promises {
		data, err := p.Await()
		res[i] = Result[T]{
			Data: data,
			Err:  err,
		}
	}

	return res
}

func (promises Promises[T]) AllSettledWithTimeout(timeout time.Duration) []Result[T] {
	if len(promises) == 0 {
		return []Result[T]{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return promises.AllSettledWithContext(ctx)
}

func (promises Promises[T]) AllSettledWithContext(ctx context.Context) []Result[T] {
	if len(promises) == 0 {
		return []Result[T]{}
	}

	res := make([]Result[T], len(promises))

	for i, p := range promises {
		data, err := p.AwaitWithContext(ctx)
		res[i] = Result[T]{
			Data: data,
			Err:  err,
		}
	}

	return res
}

func (promises Promises[T]) Race() (T, error) {
	if len(promises) == 0 {
		var zero T
		return zero, ErrEmptyPromises
	}

	done := make(chan Result[T], len(promises))

	for _, p := range promises {
		go func(p *Promise[T]) {
			data, err := p.Await()
			done <- Result[T]{Data: data, Err: err}
		}(p)
	}

	result := <-done
	return result.Data, result.Err
}

func (promises Promises[T]) RaceWithTimeout(timeout time.Duration) (T, error) {
	if len(promises) == 0 {
		var zero T
		return zero, ErrEmptyPromises
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return promises.RaceWithContext(ctx)
}

func (promises Promises[T]) RaceWithContext(ctx context.Context) (T, error) {
	if len(promises) == 0 {
		var zero T
		return zero, ErrEmptyPromises
	}

	done := make(chan Result[T], len(promises))

	for _, p := range promises {
		go func(p *Promise[T]) {
			data, err := p.AwaitWithContext(ctx)
			done <- Result[T]{Data: data, Err: err}
		}(p)
	}

	select {
	case result := <-done:
		return result.Data, result.Err
	case <-ctx.Done():
		var zero T
		return zero, ErrTimeout
	}
}

func (promises Promises[T]) Any() (T, error) {
	if len(promises) == 0 {
		var zero T
		return zero, ErrEmptyPromises
	}

	done := make(chan Result[T], len(promises))
	var errCount int

	for _, p := range promises {
		go func(p *Promise[T]) {
			data, err := p.Await()
			done <- Result[T]{Data: data, Err: err}
		}(p)
	}

	for range promises {
		result := <-done
		if result.Err == nil {
			return result.Data, nil
		}
		errCount++
	}

	var zero T
	return zero, errors.New("promise: all promises rejected")
}

type Result[T any] struct {
	Data T
	Err  error
}

func (r Result[T]) IsResolved() bool {
	return r.Err == nil
}

func (r Result[T]) IsRejected() bool {
	return r.Err != nil
}
