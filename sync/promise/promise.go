package promise

import (
	"context"
	"errors"
	"sync"

	"github.com/alextanhongpin/core/sync/cache"
)

type Status int

const (
	StatusPending Status = iota
	StatusFulfilled
	StatusRejected
)

type Promise[T any] struct {
	ch *Channel[*result[T]]
}

type result[T any] struct {
	Data  T
	Error error
}

type handler[T any] = func(ctx context.Context) (T, error)

func New[T any](ctx context.Context, fn handler[T]) *Promise[T] {
	p := Deferred[T](ctx)
	go func() {
		res, err := fn(p.ch.ctx)
		if err != nil {
			p.Reject(err)
		} else {
			p.Resolve(res)
		}
	}()
	return p
}

func (p *Promise[T]) Resolve(v T) {
	p.ch.Send(&result[T]{Data: v})
}

func (p *Promise[T]) Reject(err error) {
	p.ch.Send(&result[T]{Error: err})
}

func (p *Promise[T]) Abort(cause error) {
	p.ch.Close(cause)
}

func (p *Promise[T]) Await() (T, error) {
	res, err := p.ch.Recv()
	if err != nil {
		var zero T
		return zero, err
	}
	return res.Data, res.Error
}

func (p *Promise[T]) Status() Status {
	select {
	case <-p.ch.ctx.Done():
	default:
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

type indexedResult[T any] struct {
	index int
	result[T]
}

// observe owns only combinator waiters, never the promises or their work.
// Stop joins the waiters even when some inputs remain unresolved indefinitely.
func observe[T any](promises []*Promise[T]) (<-chan indexedResult[T], func()) {
	out := make(chan indexedResult[T], len(promises))
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i, p := range promises {
		wg.Go(func() {
			select {
			case <-done:
				return
			case <-p.ch.ctx.Done():
				v, err := p.Await()
				out <- indexedResult[T]{index: i, result: result[T]{Data: v, Error: err}}
			}
		})
	}
	go func() { wg.Wait(); close(out) }()
	return out, sync.OnceFunc(func() { close(done); wg.Wait() })
}

// Race returns the first observed settlement and releases its own waiters.
// It does not abort losing promises. Empty input panics.
func Race[T any](promises ...*Promise[T]) (T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ch, stop := observe(promises)
	defer stop()
	res := <-ch
	return res.Data, res.Error
}

// Any returns the first observed success, or joined errors if all inputs fail.
// Its waiters are joined before return; losing promises continue independently.
func Any[T any](promises ...*Promise[T]) (T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ch, stop := observe(promises)
	defer stop()
	var errs []error
	for res := range ch {
		if res.Error != nil {
			errs = append(errs, res.Error)
			continue
		}
		return res.Data, nil
	}
	var zero T
	return zero, errors.Join(errs...)
}

// All returns ordered results once every input succeeds, or returns immediately
// on the first observed failure. It releases waiters without aborting inputs.
func All[T any](promises ...*Promise[T]) ([]T, error) {
	if len(promises) == 0 {
		panic("no promises")
	}
	ch, stop := observe(promises)
	defer stop()
	results := make([]T, len(promises))
	for res := range ch {
		if res.Error != nil {
			return nil, res.Error
		}
		results[res.index] = res.Data
	}
	return results, nil
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

func Deferred[T any](ctx context.Context) *Promise[T] {
	return &Promise[T]{
		ch: NewChannel[*result[T]](ctx),
	}
}

func Resolve[T any](ctx context.Context, v T) *Promise[T] {
	p := Deferred[T](ctx)
	p.Resolve(v)
	return p
}

func Reject[T any](ctx context.Context, err error) *Promise[T] {
	p := Deferred[T](ctx)
	p.Reject(err)
	return p
}

func WithResolvers[T any](ctx context.Context) (p *Promise[T], resolve func(T), reject func(error)) {
	p = Deferred[T](ctx)
	return p, p.Resolve, p.Reject
}

type Map[K comparable, V any] = cache.Cache[K, Promise[V]]

func NewMap[K comparable, V any](ctx context.Context) *Map[K, V] {
	return cache.New(func(K) (*Promise[V], error) {
		d := Deferred[V](ctx)
		return d, nil
	})
}
