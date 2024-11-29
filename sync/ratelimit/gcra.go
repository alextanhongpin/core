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

func (r *GCRA) Allow() bool {
	return r.AllowN(1)
}

func (r *GCRA) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	r.last = max(r.last, now)
	if r.last-r.offset <= now {
		r.last += int64(n) * r.interval

		return true
	}

	return false
}

func (r *GCRA) RetryAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.last < now.UnixNano() {
		return now
	}

	return time.Unix(0, now.UnixNano()+r.interval)
}
