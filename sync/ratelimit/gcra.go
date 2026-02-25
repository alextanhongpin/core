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
	Now    func() time.Time
	burst  int64
	limit  int64
	period int64
}

func NewGCRA(limit int, period time.Duration, burst int) (*GCRA, error) {
	if err := validate(limit, period, burst); err != nil {
		return nil, err
	}

	return &GCRA{
		// NOTE: The burst is only applied once.
		Now:    time.Now,
		limit:  int64(limit),
		period: period.Nanoseconds(),
		burst:  int64(burst),
	}, nil
}

// MustNewGCRA creates a new GCRA rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewGCRA(limit int, period time.Duration, burst int) *GCRA {
	gcra, err := NewGCRA(limit, period, burst)
	if err != nil {
		panic(err)
	}
	return gcra
}

func (r *GCRA) Allow() bool {
	return r.AllowN(1)
}

func (r *GCRA) AllowN(n int) bool {
	return r.limitN(n).Allow
}

func (r *GCRA) Limit() *Result {
	return r.limitN(1)
}

func (r *GCRA) LimitN(n int) *Result {
	return r.limitN(n)
}

func (r *GCRA) limitN(n int) *Result {
	if n < 0 {
		return new(Result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	quantity := int64(n)
	remaining := int64(-1)
	delta := r.period / r.limit
	now := r.Now().UnixNano()
	r.last = max(r.last, now)
	if r.last-r.burst*delta <= now {
		r.last += quantity * delta
		up, lo := now+delta, r.last-r.burst*delta
		remaining = max(0, (up-lo)/delta)
	}

	retryAfter := max(0, r.last-r.burst*delta-now)

	return &Result{
		Allow:      remaining >= 0,
		Remaining:  max(0, remaining),
		RetryAfter: time.Duration(retryAfter) * time.Nanosecond,
		ResetAfter: time.Duration(retryAfter) * time.Nanosecond,
	}
}

type Result struct {
	RetryAfter time.Duration
	ResetAfter time.Duration
	Remaining  int64
	Allow      bool
}
