package promise

import (
	"context"
	"sync"
	"sync/atomic"
)

// Channel[T] is a generic single-value channel that delivers exactly one value
// or terminates with a context cancellation error. It guarantees at most one
// send via sync.Once and at most one receive via sync.OnceValues. The channel
// is bound to a context; if the context is cancelled before a value is sent,
// Recv returns context.Cause(ctx).
type Channel[T any] struct {
	ch     chan T
	read   func() (T, error)
	write  sync.Once
	done   *atomic.Bool
	cancel func(err error)
}

// NewChannel creates a new Channel[T] bound to ctx. The returned channel will
// return context.Cause(ctx) if the context is cancelled before a value is sent.
func NewChannel[T any](ctx context.Context) *Channel[T] {
	ch := make(chan T, 1)
	done := new(atomic.Bool)
	ctx, cancel := context.WithCancelCause(ctx)
	return &Channel[T]{
		done:   done,
		ch:     ch,
		cancel: cancel,
		read: sync.OnceValues(func() (T, error) {
			defer done.Store(true)

			var zero T
			select {
			case <-ctx.Done():
				return zero, context.Cause(ctx)

			case res := <-ch:
				return res, nil
			}
		}),
	}
}

// Send publishes v to the channel. It may be called only once; subsequent
// calls are no-ops due to sync.Once.
func (c *Channel[T]) Send(v T) {
	c.write.Do(func() {
		c.ch <- v
	})
}

// Recv blocks until a value is sent or the context is cancelled, then returns
// the value and any error. The operation is executed at most once.
func (c *Channel[T]) Recv() (T, error) {
	return c.read()
}

// Close cancels the underlying context with cause, preventing further sends
// and causing Recv to return the cause error if no value was sent. It may be
// called only once.
func (c *Channel[T]) Close(cause error) {
	c.write.Do(func() {
		c.cancel(cause)
	})
}

// Done reports whether Recv has completed.
func (c *Channel[T]) Done() bool {
	return c.done.Load()
}
