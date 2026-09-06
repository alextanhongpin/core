package promise

import (
	"context"
	"errors"
	"sync/atomic"
)

var errDone = errors.New("done")

// Channel[T] is a generic single-value channel that delivers exactly one value
// or terminates with a context cancellation error. It guarantees at most one
// send via sync.Once and at most one receive via sync.OnceValues. The channel
// is bound to a context; if the context is cancelled before a value is sent,
// Recv returns context.Cause(ctx).
type Channel[T any] struct {
	cancel func(err error)
	ctx    context.Context
	p      atomic.Pointer[T]
	zero   *T
}

// NewChannel creates a new Channel[T] bound to ctx. The returned channel will
// return context.Cause(ctx) if the context is cancelled before a value is sent.
func NewChannel[T any](ctx context.Context) *Channel[T] {
	ctx, cancel := context.WithCancelCause(ctx)
	ch := &Channel[T]{
		ctx:    ctx,
		cancel: cancel,
	}
	ch.zero = ch.p.Load()
	return ch
}

// Send publishes v to the channel. It may be called only once; subsequent
// calls are no-ops due to sync.Once.
func (c *Channel[T]) Send(v T) {
	if c.p.CompareAndSwap(c.zero, &v) {
		c.cancel(errDone)
	}
}

// Recv blocks until a value is sent or the context is cancelled, then returns
// the value and any error. The operation is executed at most once.
func (c *Channel[T]) Recv() (T, error) {
	<-c.ctx.Done()

	err := context.Cause(c.ctx)
	if !errors.Is(err, errDone) {
		var zero T
		return zero, err
	}

	return *c.p.Load(), nil
}

// Close cancels the underlying context with cause, preventing further sends
// and causing Recv to return the cause error if no value was sent. It may be
// called only once.
func (c *Channel[T]) Close(cause error) {
	c.cancel(cause)
}

// Done reports whether Recv has completed.
func (c *Channel[T]) Done() bool {
	return c.p.Load() != c.zero
}
