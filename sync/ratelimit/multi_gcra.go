package ratelimit

import (
	"sync"
	"time"
)

type MultiGCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

	// Option.
	offset   int64
	interval int64
	Now      func() time.Time
}

func NewMultiGCRA(limit int, period time.Duration, burst int) *MultiGCRA {
	interval := period.Nanoseconds() / int64(limit)

	return &MultiGCRA{
		// NOTE: The burst is only applied once.
		state:    make(map[string]int64),
		offset:   interval * int64(burst),
		interval: interval,
		Now:      time.Now,
	}
}

func (g *MultiGCRA) Allow(key string) bool {
	return g.AllowN(key, 1)
}

func (g *MultiGCRA) AllowN(key string, n int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.Now().UnixNano()
	g.state[key] = max(g.state[key], now)
	if g.state[key]-g.offset <= now {
		g.state[key] += int64(n) * g.interval

		return true
	}

	return false
}

func (g *MultiGCRA) RetryAt(key string) time.Time {
	g.mu.RLock()
	last := g.state[key]
	interval := g.interval
	g.mu.RUnlock()

	now := g.Now()
	if last < now.UnixNano() {
		return now
	}

	return time.Unix(0, now.UnixNano()+interval)
}
