package goroutine

import (
	"context"
	"sync"
)

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
}

func (g *Goroutine) stop() {
	if g.cancel == nil {
		return
	}
	g.cancel()
	g.cancel = nil
	g.wg.Wait()
}

func (g *Goroutine) start(ctx context.Context, fn func(context.Context)) {
	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	g.wg.Go(func() {
		fn(ctx)
	})
}
