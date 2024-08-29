package promise

import (
	"sync"
)

type Pool[T any] struct {
	wg       sync.WaitGroup
	limit    int
	promises Promises[T]
	sem      chan struct{}
}

func NewPool[T any](limit int) *Pool[T] {
	sem := make(chan struct{}, limit)
	for range limit {
		sem <- struct{}{}
	}

	return &Pool[T]{
		limit: limit,
		sem:   sem,
	}
}

func (p *Pool[T]) Do(fn func() (T, error)) {
	<-p.sem
	p.wg.Add(1)

	pn := New(fn)
	p.promises = append(p.promises, pn)

	go func() {
		defer func() {
			p.sem <- struct{}{}
		}()
		defer p.wg.Done()

		pn.Await()
	}()
}

func (p *Pool[T]) All() ([]T, error) {
	p.wg.Wait()

	return p.promises.All()
}

func (p *Pool[T]) AllSettled() []Result[T] {
	p.wg.Wait()

	return p.promises.AllSettled()
}
