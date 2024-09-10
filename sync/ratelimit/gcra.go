package ratelimit

import (
	"sync"
	"time"
)

type GCRA struct {
	// State.
	mu sync.RWMutex
	ts int64

	// Option.
	offset   int64
	interval int64
	Now      func() time.Time
}

func NewGCRA(limit int64, period time.Duration, burst int64) *GCRA {
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
	if g.ts < now {
		g.ts = now
	}

	if g.ts-g.offset <= now {
		g.ts += int64(n) * g.interval

		return true
	}

	return false
}

func (g *GCRA) RetryAfter() time.Duration {
	g.mu.RLock()
	ts := g.ts
	offset := g.offset
	g.mu.RUnlock()

	now := g.Now().UnixNano()
	return time.Duration(max(0, ts-offset-now))
}
