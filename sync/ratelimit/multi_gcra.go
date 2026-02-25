package ratelimit

import (
	"sync"
	"time"
)

type MultiGCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

	// Option.
	Now    func() time.Time
	burst  int64
	limit  int64
	period int64
}

func NewMultiGCRA(limit int, period time.Duration, burst int) (*MultiGCRA, error) {
	if err := validate(limit, period, burst); err != nil {
		return nil, err
	}

	return &MultiGCRA{
		// NOTE: The burst is only applied once.
		Now:    time.Now,
		burst:  int64(burst),
		limit:  int64(limit),
		period: period.Nanoseconds(),
		state:  make(map[string]int64),
	}, nil
}

// MustNewMultiGCRA creates a new multi GCRA rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewMultiGCRA(limit int, period time.Duration, burst int) *MultiGCRA {
	mgcra, err := NewMultiGCRA(limit, period, burst)
	if err != nil {
		panic(err)
	}
	return mgcra
}

func (r *MultiGCRA) Allow(key string) bool {
	return r.AllowN(key, 1)
}

func (r *MultiGCRA) AllowN(key string, n int) bool {
	return r.limitN(key, 1).Allow
}

func (r *MultiGCRA) LimitN(key string, n int) *Result {
	return r.limitN(key, 1)
}

func (r *MultiGCRA) limitN(key string, n int) *Result {
	if key == "" || n < 0 {
		return new(Result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	quantity := int64(n)
	remaining := int64(-1)
	delta := r.period / r.limit
	now := r.Now().UnixNano()

	last := r.state[key]
	last = max(last, now)
	if last-r.burst*delta <= now {
		last += quantity * delta
		up, lo := now+delta, last-r.burst*delta
		remaining = max(0, (up-lo)/delta)
	}
	r.state[key] = last

	retryAfter := max(0, last-r.burst*delta-now)

	return &Result{
		Allow:      remaining >= 0,
		Remaining:  max(0, remaining),
		RetryAfter: time.Duration(retryAfter) * time.Nanosecond,
		ResetAfter: time.Duration(retryAfter) * time.Nanosecond,
	}
}

func (r *MultiGCRA) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}

func (r *MultiGCRA) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
