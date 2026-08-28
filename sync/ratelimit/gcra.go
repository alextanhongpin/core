package ratelimit

import (
	"cmp"
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

func NewGCRA(cfg *Config) *GCRA {
	cfg = cmp.Or(cfg, DefaultConfig())
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
	delta := r.period / r.limit
	now := time.Now().UnixNano()

	last := max(r.state[key], now)
	allow := last-r.burst*delta <= now
	if allow {
		last += quantity * delta
	}
	r.state[key] = last

	var retryAfter int64
	if !allow {
		retryAfter = last - r.burst*delta - now
	}
	remaining := int64(0)
	if allow {
		up := now + delta
		lo := last - r.burst*delta
		rem := max((up-lo)/delta, 0)
		remaining = rem
	}

	return &Result{
		Allow:      allow,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(max(0, retryAfter)),
		ResetAfter: time.Duration(max(0, retryAfter)),
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
