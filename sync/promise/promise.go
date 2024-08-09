package promise

import (
	"errors"
	"sync"
)

type Promise[T any] struct {
	wg   sync.WaitGroup
	once sync.Once
	data T
	err  error
}

func Deferred[T any]() *Promise[T] {
	p := &Promise[T]{}
	p.wg.Add(1)
	return p
}

func Resolve[T any](v T) *Promise[T] {
	p := &Promise[T]{}
	p.wg.Add(1)
	return p.Resolve(v)
}

func Reject[T any](err error) *Promise[T] {
	p := &Promise[T]{}
	p.wg.Add(1)
	return p.Reject(err)
}

func New[T any](fn func() (T, error)) *Promise[T] {
	p := &Promise[T]{}
	p.wg.Add(1)

	go func() {
		v, err := fn()
		if err != nil {
			p.Reject(err)
		} else {
			p.Resolve(v)
		}
	}()

	return p
}

func (p *Promise[T]) Resolve(v T) *Promise[T] {
	p.once.Do(func() {
		p.data = v
		p.wg.Done()
	})
	return p
}

func (p *Promise[T]) Reject(err error) *Promise[T] {
	p.once.Do(func() {
		p.err = err
		p.wg.Done()
	})
	return p
}

func (p *Promise[T]) Result() Result[T] {
	p.wg.Wait()

	return Result[T]{Data: p.data, Err: p.err}
}

func (p *Promise[T]) Await() (T, error) {
	p.wg.Wait()

	return p.data, p.err
}

type Promises[T any] []*Promise[T]

func (promises Promises[T]) All() ([]T, error) {
	res := make([]T, len(promises))

	var errs error
	for i, p := range promises {
		v, err := p.Await()
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			res[i] = v
		}
	}
	if errs != nil {
		return nil, errs
	}

	return res, nil
}

func (promises Promises[T]) AllSettled() []Result[T] {
	res := make([]Result[T], len(promises))

	for i, p := range promises {
		res[i] = p.Result()
	}

	return res
}

type Result[T any] struct {
	Data T
	Err  error
}
