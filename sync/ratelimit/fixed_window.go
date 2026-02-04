package ratelimit

import (
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	last  int64
	count int

	// Options.
	limit  int
	period int64
	Now    func() time.Time
}

func NewFixedWindow(limit int, period time.Duration) (*FixedWindow, error) {
	o := &option{limit: limit, period: period}
	if err := o.Validate(); err != nil {
		return nil, err
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
		return false
	}

	now := r.Now().UnixNano()
	if r.last+r.period <= now {
		r.last = now
		r.count = 0
	}
	if r.count+n <= r.limit {
		r.count += n
		return true
	}

	return false
}

func (r *FixedWindow) Remaining() int {
	if r.last+r.period <= r.Now().UnixNano() {
		return r.limit
	}

	return r.limit - r.count
}

func (r *FixedWindow) RetryAt() time.Time {
	now := r.Now()
	if r.last+r.period <= now.UnixNano() {
		return now
	}

	if r.limit-r.count > 0 {
		return now
	}

	return time.Unix(0, r.last+r.period)
}
