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
	state map[string]*fixedWindowState
	// Options.
	limit  int
	period int64
	Now    func() time.Time
}

func NewMultiFixedWindow(limit int, period time.Duration) *MultiFixedWindow {
	return &MultiFixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		state:  make(map[string]*fixedWindowState),
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

	if r.remaining(key) >= n {
		r.add(key, n)
		return true
	}

	return false
}

func (r *MultiFixedWindow) Remaining(key string) int {
	r.mu.RLock()
	n := r.remaining(key)
	r.mu.RUnlock()

	return n
}

func (r *MultiFixedWindow) RetryAt(key string) time.Time {
	if r.Remaining(key) > 0 {
		return r.Now()
	}

	r.mu.RLock()
	v, ok := r.state[key]
	if !ok {
		r.mu.RUnlock()

		return r.Now()
	}
	nsec := v.last + r.period
	r.mu.RUnlock()

	return time.Unix(0, nsec)
}

func (r *MultiFixedWindow) remaining(key string) int {
	v, ok := r.state[key]
	if !ok {
		return r.limit
	}

	now := r.Now().UnixNano()
	if v.last+r.period <= now {
		return r.limit
	}

	return r.limit - v.count
}

func (r *MultiFixedWindow) add(key string, n int) {
	v, ok := r.state[key]
	if !ok {
		v = new(fixedWindowState)
		r.state[key] = v
	}

	now := r.Now().UnixNano()
	if v.last+r.period <= now {
		v.count = 0
		v.last = now
	}

	v.count += n
}
