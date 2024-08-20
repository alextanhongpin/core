package promise

import (
	"sync"
	"sync/atomic"
)

type Status int64

const (
	NotStarted Status = iota
	Pending
	Fulfilled
	Rejected
)

func (s Status) Int64() int64 {
	return int64(s)
}

type Promise[T any] struct {
	wg     sync.WaitGroup
	once   sync.Once
	data   T
	err    error
	status atomic.Int64
}

func Deferred[T any]() *Promise[T] {
	p := &Promise[T]{}
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
	p := Deferred[T]()

	go func() {
		p.Wait(fn)
	}()

	return p
}

func (p *Promise[T]) Resolve(v T) *Promise[T] {
	p.once.Do(func() {
		p.data = v
		p.wg.Done()
		p.status.Store(Fulfilled.Int64())
	})
	return p
}

func (p *Promise[T]) Reject(err error) *Promise[T] {
	p.once.Do(func() {
		p.err = err
		p.wg.Done()
		p.status.Store(Rejected.Int64())
	})
	return p
}

func (p *Promise[T]) Result() Result[T] {
	p.wg.Wait()

	return Result[T]{Data: p.data, Err: p.err}
}

func (p *Promise[T]) Wait(fn func() (T, error)) (T, error) {
	if p.status.CompareAndSwap(NotStarted.Int64(), Pending.Int64()) {
		v, err := fn()
		if err != nil {
			p.Reject(err)
		} else {
			p.Resolve(v)
		}
	}

	return p.Await()
}

func (p *Promise[T]) Await() (T, error) {
	p.wg.Wait()

	return p.data, p.err
}

func (p *Promise[T]) Status() Status {
	return Status(p.status.Load())
}

type Promises[T any] []*Promise[T]

func (promises Promises[T]) All() ([]T, error) {
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
