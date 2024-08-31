// package snapshot implements redis-snapshot like mechanism - the higher the
// frequency, the more frequent the execution.
package snapshot

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrTerminated = errors.New("snapshot: terminated")

type Event struct {
	Count  int
	Policy Policy
}

type Policy struct {
	Every    int
	Interval time.Duration
}

func NewOptions() []Policy {
	return []Policy{
		{Every: 1_000, Interval: time.Second},
		{Every: 100, Interval: 10 * time.Second},
		{Every: 10, Interval: time.Minute},
		{Every: 1, Interval: time.Hour},
	}
}

type Background struct {
	opts []Policy
	ch   chan int
	ctx  context.Context
	fn   func(ctx context.Context, evt Event)
}

func New(ctx context.Context, fn func(context.Context, Event), opts ...Policy) (*Background, func()) {
	bg := &Background{
		opts: opts,
		ch:   make(chan int),
		fn:   fn,
	}
	ctx, cancel := context.WithCancelCause(ctx)
	stop := bg.init(ctx)
	bg.ctx = ctx

	return bg, func() {
		cancel(ErrTerminated)
		stop()
	}
}

func (b *Background) Inc(n int) error {
	select {
	case <-b.ctx.Done():
		return context.Cause(b.ctx)
	case b.ch <- n:
		return nil
	}
}

func (b *Background) init(ctx context.Context) func() {
	var count int

	var wg sync.WaitGroup
	wg.Add(len(b.opts))

	ch := make(chan Policy)

	for _, p := range b.opts {
		go func() {
			defer wg.Done()

			t := time.NewTicker(p.Interval)
			defer t.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					select {
					case ch <- p:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case n := <-b.ch:
				count += n
			case p := <-ch:
				if count < p.Every {
					continue
				}
				evt := Event{
					Count:  count,
					Policy: p,
				}
				count = 0
				b.fn(ctx, evt)
			}
		}
	}()

	return wg.Wait
}
