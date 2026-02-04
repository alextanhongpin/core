package ratelimit

import (
	"fmt"
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

func NewFixedWindow(limit int, period time.Duration) (*FixedWindow, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit", ErrInvalidNumber)
	}
	if period <= 0 {
		return nil, fmt.Errorf("%w: period", ErrInvalidNumber)
	}

	return &FixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}, nil
}

// MustNewFixedWindow creates a new fixed window rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewFixedWindow(limit int, period time.Duration) *FixedWindow {
	fw, err := NewFixedWindow(limit, period)
	if err != nil {
		panic(err)
	}
	return fw
}

func (r *FixedWindow) Allow() bool {
	return r.AllowN(1)
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (r *FixedWindow) AllowN(n int) bool {
	if n <= 0 {
		// TODO:
		// panic(ErrInvalidNumber)
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.clear(r.Now())
	if r.remaining() >= n {
		r.count += n

		return true
	}

	return false
}

func (r *FixedWindow) Remaining() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.expired(r.Now()) {
		return r.limit
	}

	return r.remaining()
}

func (r *FixedWindow) RetryAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.expired(now) {
		return now
	}

	if r.remaining() > 0 {
		return now
	}

	nsec := r.last + r.period
	return time.Unix(0, nsec)
}

func (r *FixedWindow) remaining() int {
	return r.limit - r.count
}

func (r *FixedWindow) expired(at time.Time) bool {
	return r.last+r.period <= at.UnixNano()
}

func (r *FixedWindow) clear(at time.Time) {
	if r.expired(at) {
		r.count = 0
		r.last = at.UnixNano()
	}
}
