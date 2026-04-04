package pool

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type shortLived[T any] struct {
	cleanup func()
	once    sync.Once
	val     T
	wg      sync.WaitGroup
}

func (s *shortLived[T]) Cleanup() {
	s.once.Do(func() {
		s.wg.Wait()
		s.cleanup()
	})
}

func newShortLived[T any](v T, cleanup func()) *shortLived[T] {
	return &shortLived[T]{
		cleanup: cleanup,
		val:     v,
	}
}

type Pool[T any] struct {
	New       func() (T, func())
	done      bool
	mu        sync.Mutex
	sometimes rate.Sometimes
	val       *shortLived[T]
	wg        sync.WaitGroup
}

// New return an instance of an object that is valid for n-usage and
// t-duration.
func New[T any](newFn func() (T, func()), every int, interval time.Duration) *Pool[T] {
	return &Pool[T]{
		New: newFn,
		sometimes: rate.Sometimes{
			Every:    every,
			Interval: interval,
		},
	}
}

func (p *Pool[T]) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.done {
		p.done = true
		p.wg.Wait()

		old := p.val
		old.Cleanup()
	}
}

func (p *Pool[T]) Borrow() (T, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.done {
		p.sometimes.Do(func() {
			old := p.val
			p.val = newShortLived(p.New())
			if old == nil {
				return
			}
			p.wg.Go(old.Cleanup)
		})
	}
	curr := p.val
	curr.wg.Add(1)
	return curr.val, curr.wg.Done
}
