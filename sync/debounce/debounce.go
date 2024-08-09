package debounce

import (
	"sync"
	"time"
)

type Group struct {
	Every   int
	Timeout time.Duration
	count   int
	t       *time.Timer
	mu      sync.Mutex
}

func (g *Group) Do(fn func()) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.t != nil {
		g.t.Stop()
		g.t = nil
	}
	if g.Every > 0 && g.count > 0 && g.count%g.Every == 0 {
		g.count = 0

		fn()
		return
	}

	g.t = time.AfterFunc(g.Timeout, fn)
	g.count++
}
