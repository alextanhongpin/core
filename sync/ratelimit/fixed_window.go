package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	mu    sync.RWMutex
	last  int64
	count int

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

	now := r.Now()
	if r.isExpired(now) {
		r.count = 0
		r.last = now.UnixNano()
	}

	if r.limit-r.count >= n {
		r.count += n

		return true
	}

	return false
}

func (r *FixedWindow) Remaining() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isExpired(r.Now()) {
		return r.limit
	}

	return r.limit - r.count
}

func (r *FixedWindow) RetryAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.isExpired(now) {
		return now
	}

	if r.limit-r.count > 0 {
		return now
	}

	nsec := r.last + r.period
	return time.Unix(0, nsec)
}

func (r *FixedWindow) isExpired(at time.Time) bool {
	return r.last+r.period <= at.UnixNano()
}
