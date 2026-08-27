package ratelimit

import (
	"sync"
	"time"
)

var _ ratelimiter = (*GCRA)(nil)

type GCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

	// Option.
	burst  int64
	limit  int64
	period int64
}

func NewGCRA(cfg Config) *GCRA {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	return &GCRA{
		burst:  int64(cfg.Burst),
		limit:  int64(cfg.Limit),
		period: cfg.Period.Nanoseconds(),
		state:  make(map[string]int64),
	}
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
	now := time.Now().UnixNano()

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
		RetryAfter: time.Duration(retryAfter),
		ResetAfter: time.Duration(retryAfter),
		Limit:      int(r.limit),
	}
}

func (r *GCRA) Clear() {
	r.mu.Lock()
	now := time.Now().UnixNano()
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
