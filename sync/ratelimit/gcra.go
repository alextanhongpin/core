package ratelimit

import (
	"sync"
	"time"
)

type GCRA struct {
	// State.
	mu   sync.RWMutex
	last int64

	// Option.
	offset   int64
	interval int64
	Now      func() time.Time
}

func NewGCRA(limit int, period time.Duration, burst int) *GCRA {
	interval := period.Nanoseconds() / int64(limit)

	return &GCRA{
		// NOTE: The burst is only applied once.
		offset:   interval * int64(burst),
		interval: interval,
		Now:      time.Now,
	}
}

func (g *GCRA) Allow() bool {
	return g.AllowN(1)
}

func (g *GCRA) AllowN(n int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.Now().UnixNano()
	g.last = max(g.last, now)
	if g.last-g.offset <= now {
		g.last += int64(n) * g.interval

		return true
	}

	return false
}

func (g *GCRA) RetryAt() time.Time {
	g.mu.RLock()
	last := g.last
	interval := g.interval
	g.mu.RUnlock()

	now := g.Now()
	if last < now.UnixNano() {
		return now
	}

	return time.Unix(0, now.UnixNano()+interval)
}
