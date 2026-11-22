package broadcast

import (
	"slices"
	"sync"
)

// Broadcast delivers values serially to registered subscribers. Slow subscribers
// apply backpressure. Construct it with New; the zero value is not usable.
// Values are shared without copying; callers own their synchronization.
type Broadcast[T any] struct {
	ch       chan T
	done     chan struct{}
	register chan *subscriber[T]
	wg       sync.WaitGroup
	mu       sync.Mutex
}

// subscriber is one registered receiver. mu is held by the dispatcher while it
// delivers to ch, and by unsubscribe/shutdown while closing ch, so ch is never
// closed during a send. quit unblocks a delivery in progress before mu is taken.
type subscriber[T any] struct {
	ch     chan T
	quit   chan struct{}
	mu     sync.Mutex
	closed bool
}

func (s *subscriber[T]) close() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.ch)
	}
	s.mu.Unlock()
}

// New starts a dispatcher. The returned stop function is concurrent-safe and
// idempotent, interrupts pending delivery, closes subscriber channels, and waits
// for worker callbacks. Callbacks must not call stop, Send, Chan, Subscribe, or
// Go on this instance: those operations can wait for the callback itself to
// finish.
func New[T any]() (*Broadcast[T], func()) {
	mu := &Broadcast[T]{
		ch:       make(chan T),
		done:     make(chan struct{}),
		register: make(chan *subscriber[T]),
	}

	mu.wg.Go(func() {
		var subs []*subscriber[T]

		defer func() {
			for _, s := range subs {
				s.close()
			}
		}()

		// prune drops subscribers that have unsubscribed.
		prune := func() {
			subs = slices.DeleteFunc(subs, func(s *subscriber[T]) bool {
				select {
				case <-s.quit:
					return true
				default:
					return false
				}
			})
		}

		for {
			select {
			case <-mu.done:
				return

			case s := <-mu.register:
				prune()
				subs = append(subs, s)

			case v := <-mu.ch:
				prune()
				for _, s := range subs {
					s.mu.Lock()
					if s.closed {
						s.mu.Unlock()
						continue
					}
					select {
					case s.ch <- v:
					case <-s.quit:
					case <-mu.done:
						s.mu.Unlock()
						return
					}
					s.mu.Unlock()
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

// Chan registers a subscriber and returns its channel. The subscription lasts
// until stop; use Subscribe to be able to leave earlier. Registration waits for
// the dispatcher, so it can block behind a slow subscriber. After stop, Chan
// returns a closed channel.
func (b *Broadcast[T]) Chan() <-chan T {
	ch, _ := b.Subscribe()
	return ch
}

// Subscribe registers a subscriber and returns its channel and an unsubscribe
// function. Unsubscribe is idempotent and safe to call concurrently, including
// while a Send is delivering to, or blocked on, this subscriber: the subscriber
// is removed and its channel closed without waiting for the dispatcher. Values
// already received stay valid; values not yet delivered to it are dropped.
// Registration waits for the dispatcher, so it can block while a slow
// subscriber applies backpressure. After stop, Subscribe returns a closed
// channel and a no-op unsubscribe. Do not call it from a Go callback.
func (b *Broadcast[T]) Subscribe() (<-chan T, func()) {
	s := &subscriber[T]{ch: make(chan T), quit: make(chan struct{})}

	select {
	case <-b.done:
		close(s.ch)
		return s.ch, func() {}

	case b.register <- s:
		return s.ch, sync.OnceFunc(func() {
			close(s.quit) // unblocks a delivery holding s.mu
			s.close()
		})
	}
}

// Go runs fn for each value on its own goroutine until stop. Registration
// waits for the dispatcher, like Chan. After stop it does nothing.
func (b *Broadcast[T]) Go(fn func(T)) {
	s := &subscriber[T]{ch: make(chan T), quit: make(chan struct{})}
	ch := s.ch

	select {
	case <-b.done:
		return

	case b.register <- s:
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
