package goroutine

import (
	"context"
	"sync"
)

// Goroutine owns cancelable work. Its zero value is ready to use.
// Start cancels previous work without waiting; callbacks may overlap.
// Stop waits for all callbacks, which must cooperate with cancellation and
// must not call Start or Stop on this instance while Stop is waiting.
type Goroutine struct {
	mu     sync.Mutex
	cancel func()
	wg     sync.WaitGroup
}

func New() *Goroutine {
	return &Goroutine{}
}

func (g *Goroutine) Start(ctx context.Context, fn func(context.Context)) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.stop()
	g.start(ctx, fn)
}

func (g *Goroutine) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.stop()
	g.wg.Wait()
}

func (g *Goroutine) start(ctx context.Context, fn func(context.Context)) {
	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	g.wg.Go(func() {
		defer cancel()
		fn(ctx)
	})
}

func (g *Goroutine) stop() {
	if g.cancel == nil {
		return
	}
	g.cancel()
	g.cancel = nil
}
