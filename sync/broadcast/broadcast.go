package broadcast

import (
	"sync"
)

// Broadcast delivers values serially to registered subscribers. Slow subscribers
// apply backpressure. Construct it with New; the zero value is not usable.
// Values are shared without copying; callers own their synchronization.
type Broadcast[T any] struct {
	ch       chan T
	done     chan struct{}
	register chan chan T
	wg       sync.WaitGroup
	mu       sync.Mutex
}

// New starts a dispatcher. The returned stop function is concurrent-safe and
// idempotent, interrupts pending delivery, closes subscriber channels, and waits
// for worker callbacks. Callbacks must not call stop, Send, Chan, or Go on this
// instance: those operations can wait for the callback itself to finish.
func New[T any]() (*Broadcast[T], func()) {
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
					select {
					case ch <- v:
					case <-mu.done:
						return
					}
				}
			}
		}
	})

	return mu, sync.OnceFunc(func() {
		mu.mu.Lock()
		close(mu.done)
		mu.mu.Unlock()
		mu.wg.Wait()
	})
}

// Send blocks until the dispatcher accepts n or shutdown begins. Acceptance
// does not guarantee delivery: stop may interrupt delivery to any subscriber.
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
		b.mu.Lock()
		defer b.mu.Unlock()
		select {
		case <-b.done:
			return
		default:
		}
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
