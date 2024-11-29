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

func (r *MultiGCRA) Allow(key string) bool {
	return r.AllowN(key, 1)
}

func (r *MultiGCRA) AllowN(key string, n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	r.state[key] = max(r.state[key], now)
	if r.state[key]-r.offset <= now {
		r.state[key] += int64(n) * r.interval

		return true
	}

	return false
}

func (r *MultiGCRA) RetryAt(key string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.state[key] < now.UnixNano() {
		return now
	}

	return time.Unix(0, r.state[key]+r.interval)
}
