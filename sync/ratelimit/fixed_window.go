package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	mu sync.RWMutex
	// State.
	last      int64
	remaining int64

	// Options.
	limit  int64
	period int64
	Now    func() time.Time
}

func NewFixedWindow(limit int, period time.Duration) (*FixedWindow, error) {
	if err := validate(limit, period, 0); err != nil {
		return nil, err
	}

	return &FixedWindow{
		limit:  int64(limit),
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
	return r.LimitN(n).Allow
}

func (r *FixedWindow) Limit() *Result {
	return r.LimitN(1)
}
func (r *FixedWindow) LimitN(n int) *Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	if r.last+r.period <= now {
		r.last = now
		r.remaining = r.limit
	}
	r.remaining = max(-1, r.remaining-1)

	res := &Result{
		Allow:      r.remaining >= 0,
		Remaining:  max(0, r.remaining),
		ResetAfter: time.Duration(now-(r.last+r.period)) * time.Nanosecond,
		RetryAfter: time.Duration(now-(r.last+r.period)) * time.Nanosecond,
	}
	if res.Allow {
		res.RetryAfter = 0
	}
	return res
}
