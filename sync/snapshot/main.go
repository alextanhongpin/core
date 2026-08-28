package snapshot

import (
	"sync"
)

type Broadcast[T any] struct {
	ch       chan T
	done     chan struct{}
	register chan chan T
	wg       sync.WaitGroup
}

func NewBroadcast[T any]() (*Broadcast[T], func()) {
	mu := &Broadcast[T]{
		ch:       make(chan T),
		done:     make(chan struct{}),
		register: make(chan chan T),
	}

	mu.wg.Go(func() {
		var chans []chan T
		defer func() {
			for _, ch := range chans {
				close(ch)
			}
		}()
		for {
			select {
			case <-mu.done:
				return

			case ch := <-mu.register:
				chans = append(chans, ch)

			case v := <-mu.ch:
				for _, ch := range chans {
					ch <- v
				}
			}
		}
	})

	return mu, sync.OnceFunc(func() {
		close(mu.done)
		mu.wg.Wait()
	})
}

func (b *Broadcast[T]) Send(n T) {
	select {
	case <-b.done:
		return
	case b.ch <- n:
	}
}

func (b *Broadcast[T]) Chan() <-chan T {
	ch := make(chan T)
	select {
	case <-b.done:
		return nil
	case b.register <- ch:
		return ch
	}
}

func (b *Broadcast[T]) Go(fn func(T)) {
	ch := make(chan T)
	select {
	case <-b.done:
		return
	case b.register <- ch:
		b.wg.Go(func() {
			for {
				select {
				case v, ok := <-ch:
					if !ok {
						return
					}
					fn(v)
				case <-b.done:
					return
				}
			}
		})
	}
}
