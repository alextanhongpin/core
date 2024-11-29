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
	interval int64
	offset   int64
	period   int64
	Now      func() time.Time
}

func NewMultiGCRA(limit int, period time.Duration, burst int) *MultiGCRA {
	interval := period.Nanoseconds() / int64(limit)

	return &MultiGCRA{
		// NOTE: The burst is only applied once.
		state:    make(map[string]int64),
		interval: interval,
		offset:   interval * int64(burst),
		period:   period.Nanoseconds(),
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

func (r *MultiGCRA) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}
