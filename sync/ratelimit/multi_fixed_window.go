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

func NewMultiFixedWindow(limit int, period time.Duration) (*MultiFixedWindow, error) {
	o := &option{limit: limit, period: period}
	if err := o.Validate(); err != nil {
		return nil, err
	}

	return &MultiFixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		state:  make(map[string]fixedWindowState),
		Now:    time.Now,
	}, nil
}

// MustNewMultiFixedWindow creates a new multi fixed window rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewMultiFixedWindow(limit int, period time.Duration) *MultiFixedWindow {
	mfw, err := NewMultiFixedWindow(limit, period)
	if err != nil {
		panic(err)
	}
	return mfw
}

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (r *MultiFixedWindow) Allow(key string) bool {
	return r.AllowN(key, 1)
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (r *MultiFixedWindow) AllowN(key string, n int) bool {
	if key == "" || n <= 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now()
	if r.isExpired(key, now) {
		r.state[key] = fixedWindowState{count: 0, last: now.UnixNano()}
	}

	s := r.state[key]
	if r.limit-s.count >= n {
		s.count += n
		r.state[key] = s

		return true
	}

	return false
}

func (r *MultiFixedWindow) Remaining(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isExpired(key, r.Now()) {
		return r.limit
	}

	return r.limit - r.state[key].count
}

func (r *MultiFixedWindow) RetryAt(key string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.isExpired(key, now) {
		return now
	}

	s := r.state[key]
	if r.limit > s.count {
		return now
	}

	nsec := s.last + r.period
	return time.Unix(0, nsec)
}

func (r *MultiFixedWindow) isExpired(key string, at time.Time) bool {
	return r.state[key].last+r.period <= at.UnixNano()
}

func (r *MultiFixedWindow) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v.last+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}

func (r *MultiFixedWindow) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
