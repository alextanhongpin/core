package ratelimit

import (
	"math"
	"sync"
	"time"
)

type slidingWindowState struct {
	prev   int
	curr   int
	window int64
}

type MultiSlidingWindow struct {
	// State.
	mu    sync.RWMutex
	state map[string]slidingWindowState

	// Options.
	limit  int
	period int64
	Now    func() time.Time
}

func NewMultiSlidingWindow(limit int, period time.Duration) *MultiSlidingWindow {
	return &MultiSlidingWindow{
		// NOTE: The burst is only applied once.
		state:  make(map[string]slidingWindowState),
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}
}

func (r *MultiSlidingWindow) Allow(key string) bool {
	return r.AllowN(key, 1)
}

func (r *MultiSlidingWindow) AllowN(key string, n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.remaining(key) >= n {
		r.add(key, n)

		return true
	}

	return false
}

func (r *MultiSlidingWindow) Remaining(key string) int {
	r.mu.RLock()
	n := r.remaining(key)
	r.mu.RUnlock()

	return n
}

func (r *MultiSlidingWindow) remaining(key string) int {
	now := r.Now().UnixNano()
	window := now - now%r.period

	s := r.state[key]
	prev := s.prev
	curr := s.curr
	if s.window == window-r.period {
		prev = s.curr
		curr = 0
	} else if s.window != window {
		prev = 0
		curr = 0
	}

	ratio := 1 - float64(now-window)/float64(r.period)

	return r.limit - (int(math.Ceil(ratio*float64(prev))) + curr)
}

func (r *MultiSlidingWindow) add(key string, n int) {
	now := r.Now().UnixNano()
	window := now - now%r.period

	s := r.state[key]

	if s.window == window-r.period {
		s.prev = s.curr
		s.curr = 0
		s.window = window
	} else if s.window != window {
		s.prev = 0
		s.curr = 0
		s.window = window
	}

	s.curr += n
	r.state[key] = s
}

func (r *MultiSlidingWindow) Clear() {
	r.mu.Lock()
	now := r.Now().UnixNano()
	for k, v := range r.state {
		if v.window+r.period <= now {
			delete(r.state, k)
		}
	}
	r.mu.Unlock()
}
