package ratelimit

import (
	"sync"
	"time"
)

type fixedWindowState struct {
	count int
	last  int64
}

// MultiFixedWindow acts as a counter for a given time period.
type MultiFixedWindow struct {
	// State.
	mu    sync.RWMutex
	state map[string]fixedWindowState
	// Options.
	limit  int
	period int64
	Now    func() time.Time
}

func NewMultiFixedWindow(limit int, period time.Duration) *MultiFixedWindow {
	return &MultiFixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		state:  make(map[string]fixedWindowState),
		Now:    time.Now,
	}
}

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (r *MultiFixedWindow) Allow(key string) bool {
	return r.AllowN(key, 1)
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (r *MultiFixedWindow) AllowN(key string, n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.snapshot(r.Now(), key)
	if r.limit-s.count >= n {
		s.count += n
		r.state[key] = s

		return true
	}

	return false
}

func (r *MultiFixedWindow) Remaining(key string) int {
	r.mu.RLock()
	s := r.snapshot(r.Now(), key)
	r.mu.RUnlock()

	return r.limit - s.count
}

func (r *MultiFixedWindow) RetryAt(key string) time.Time {
	now := r.Now()

	r.mu.RLock()
	s := r.snapshot(now, key)
	r.mu.RUnlock()

	if r.limit > s.count {
		return now
	}

	nsec := s.last + r.period
	return time.Unix(0, nsec)
}

func (r *MultiFixedWindow) snapshot(at time.Time, key string) fixedWindowState {
	now := at.UnixNano()

	s := r.state[key]
	if s.last+r.period <= now {
		return fixedWindowState{last: now}
	}

	return s
}
