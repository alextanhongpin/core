package ratelimit

import (
	"sync"
	"time"
)

type GCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

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
		Now:    time.Now,
		burst:  int64(burst),
		limit:  int64(limit),
		period: period.Nanoseconds(),
		state:  make(map[string]int64),
	}, nil
}

func (r *GCRA) Allow(key string) bool {
	return r.LimitN(key, 1).Allow
}

func (r *GCRA) AllowN(key string, n int) bool {
	// Forward the requested token count to the limiter.
	return r.LimitN(key, n).Allow
}

func (r *GCRA) Limit(key string) *Result {
	return r.LimitN(key, 1)
}

func (r *GCRA) LimitN(key string, n int) *Result {
	if key == "" || n < 0 {
		// Always invalid.
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
		Remaining:  int(max(0, remaining)),
		RetryAfter: time.Duration(retryAfter) * time.Nanosecond,
		ResetAfter: time.Duration(retryAfter) * time.Nanosecond,
	}
}

func (r *GCRA) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}

func (r *GCRA) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
