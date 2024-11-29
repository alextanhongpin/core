package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	mu    sync.RWMutex
	count int
	last  int64

	// Options.
	limit  int
	period int64
	Now    func() time.Time
}

func NewFixedWindow(limit int, period time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}
}

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (r *FixedWindow) Allow() bool {
	return r.AllowN(1)
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (r *FixedWindow) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.remaining() >= n {
		r.add(n)
		return true
	}

	return false
}

func (r *FixedWindow) Remaining() int {
	r.mu.RLock()
	n := r.remaining()
	r.mu.RUnlock()

	return n
}

func (r *FixedWindow) RetryAt() time.Time {
	if r.Remaining() > 0 {
		return r.Now()
	}

	r.mu.RLock()
	nsec := r.last + r.period
	r.mu.RUnlock()

	return time.Unix(0, nsec)
}

func (r *FixedWindow) remaining() int {
	now := r.Now().UnixNano()
	if r.last+r.period <= now {
		return r.limit
	}

	return r.limit - r.count
}

func (r *FixedWindow) add(n int) {
	now := r.Now().UnixNano()
	if r.last+r.period <= now {
		r.count = 0
		r.last = now
	}

	r.count += n
}
