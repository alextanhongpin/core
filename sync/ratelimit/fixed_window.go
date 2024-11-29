package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	mu    sync.RWMutex
	state fixedWindowState

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

	s := r.snapshot(r.Now())
	if r.limit-s.count >= n {
		s.count += n
		r.state = s

		return true
	}

	return false
}

func (r *FixedWindow) Remaining() int {
	r.mu.RLock()
	s := r.snapshot(r.Now())
	r.mu.RUnlock()

	return r.limit - s.count
}

func (r *FixedWindow) RetryAt() time.Time {
	now := r.Now()

	r.mu.RLock()
	s := r.snapshot(now)
	r.mu.RUnlock()

	if r.limit-s.count > 0 {
		return now
	}

	nsec := s.last + r.period
	return time.Unix(0, nsec)
}

func (r *FixedWindow) snapshot(at time.Time) fixedWindowState {
	now := at.UnixNano()
	if r.state.last+r.period <= now {
		return fixedWindowState{last: now}
	}

	return r.state
}
