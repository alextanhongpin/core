package ratelimit

import (
	"math"
	"sync"
	"time"
)

type slidingWindowState struct {
	curr   int64
	prev   int64
	window int64
}

func (s slidingWindowState) Sum(ratio float64) int64 {
	prev := (1.0 - ratio) * float64(s.prev)
	return int64(math.Ceil(prev)) + s.curr
}

type SlidingWindow struct {
	// State.
	mu    sync.RWMutex
	state map[string]slidingWindowState

	// Options.
	Now    func() time.Time
	limit  int64
	period int64
}

func NewSlidingWindow(limit int, period time.Duration) (*SlidingWindow, error) {
	if err := validate(limit, period, 0); err != nil {
		return nil, err
	}

	return &SlidingWindow{
		Now:    time.Now,
		limit:  int64(limit),
		period: period.Nanoseconds(),
		state:  make(map[string]slidingWindowState),
	}, nil
}

func (r *SlidingWindow) Allow(key string) bool {
	return r.LimitN(key, 1).Allow
}

func (r *SlidingWindow) AllowN(key string, n int) bool {
	return r.LimitN(key, n).Allow
}

func (r *SlidingWindow) Limit(key string) *Result {
	return r.LimitN(key, 1)
}

func (r *SlidingWindow) LimitN(key string, n int) *Result {
	if key == "" || n < 0 {
		return new(Result)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	quantity := int64(n)
	s := r.state[key]

	switch {
	case now-s.window < r.period:
		// In current window
	case now-s.window < 2*r.period:
		// In previous window.
		s.prev = s.curr
		s.curr = 0
		s.window += r.period
	default:
		s.prev = 0
		s.curr = 0
		s.window = now
	}

	ratio := float64(now-s.window) / float64(r.period)
	total := s.Sum(ratio)

	allow := false
	if total+quantity <= r.limit+1 {
		s.curr += quantity
		allow = true
	}

	r.state[key] = s

	res := &Result{
		Allow:      allow,
		Remaining:  int(max(0, r.limit-s.Sum(ratio))),
		ResetAfter: time.Duration(s.window + r.period - now),
	}
	if res.Remaining == 0 {
		res.RetryAfter = res.ResetAfter
	}
	return res
}

func (r *SlidingWindow) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v.window+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}

func (r *SlidingWindow) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
