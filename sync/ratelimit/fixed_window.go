package ratelimit

import (
	"sync"
	"time"
)

type fixedWindowState struct {
	count int64
	last  int64
}

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	mu    sync.RWMutex
	state map[string]fixedWindowState

	// Options.
	Now    func() time.Time
	limit  int64
	period int64
}

func NewFixedWindow(limit int, period time.Duration) (*FixedWindow, error) {
	if err := validate(limit, period, 0); err != nil {
		return nil, err
	}

	return &FixedWindow{
		Now:    time.Now,
		limit:  int64(limit),
		period: period.Nanoseconds(),
		state:  make(map[string]fixedWindowState),
	}, nil
}

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (r *FixedWindow) Allow(key string) bool {
	return r.LimitN(key, 1).Allow
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (r *FixedWindow) AllowN(key string, n int) bool {
	return r.LimitN(key, n).Allow
}

func (r *FixedWindow) Limit(key string) *Result {
	return r.LimitN(key, 1)
}

func (r *FixedWindow) LimitN(key string, n int) *Result {
	if key == "" || n < 0 {
		return new(Result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	curr := r.state[key]
	now := r.Now().UnixNano()
	quantity := int64(n)

	if curr.last+r.period <= now {
		curr.last = now
		curr.count = 0
	}

	if curr.count+quantity <= r.limit+1 {
		curr.count += quantity
	}

	r.state[key] = curr
	remaining := r.limit - curr.count

	res := &Result{
		Allow:      remaining >= 0,
		Remaining:  max(0, int(remaining)),
		ResetAfter: time.Duration(curr.last+r.period-now) * time.Nanosecond,
		RetryAfter: 0,
	}
	if res.Remaining == 0 {
		res.RetryAfter = res.ResetAfter
	}
	return res
}

func (r *FixedWindow) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v.last+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}

func (r *FixedWindow) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
