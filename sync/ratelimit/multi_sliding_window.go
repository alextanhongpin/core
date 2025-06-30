package ratelimit

import (
	"errors"
	"math"
	"sync"
	"time"
)

var (
	ErrInvalidMultiSlidingWindowLimit  = errors.New("multi sliding window limit must be positive")
	ErrInvalidMultiSlidingWindowPeriod = errors.New("multi sliding window period must be positive")
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

func NewMultiSlidingWindow(limit int, period time.Duration) (*MultiSlidingWindow, error) {
	if limit <= 0 {
		return nil, ErrInvalidMultiSlidingWindowLimit
	}
	if period <= 0 {
		return nil, ErrInvalidMultiSlidingWindowPeriod
	}

	return &MultiSlidingWindow{
		// NOTE: The burst is only applied once.
		state:  make(map[string]slidingWindowState),
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}, nil
}

// MustNewMultiSlidingWindow creates a new multi sliding window rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewMultiSlidingWindow(limit int, period time.Duration) *MultiSlidingWindow {
	msw, err := NewMultiSlidingWindow(limit, period)
	if err != nil {
		panic(err)
	}
	return msw
}

func (r *MultiSlidingWindow) Allow(key string) bool {
	return r.AllowN(key, 1)
}

func (r *MultiSlidingWindow) AllowN(key string, n int) bool {
	if key == "" || n <= 0 {
		return false
	}

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

	s := r.state[key]
	prev := s.prev
	curr := s.curr
	window := s.window

	if window+r.period > now {
		// In current window
	} else if window+2*r.period > now {
		// In previous window
		prev = s.curr
		curr = 0
		window += r.period
	} else {
		prev = 0
		curr = 0
		window = now
	}

	ratio := 1 - float64(now-window)/float64(r.period)

	return r.limit - (int(math.Ceil(ratio*float64(prev))) + curr)
}

func (r *MultiSlidingWindow) add(key string, n int) {
	now := r.Now().UnixNano()
	s := r.state[key]
	if s.window+r.period > now {
		// In current window
	} else if s.window+2*r.period > now {
		// In previous window
		s.prev = s.curr
		s.curr = 0
		s.window += r.period
	} else {
		s.prev = 0
		s.curr = 0
		s.window = now
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

func (r *MultiSlidingWindow) Size() int {
	r.mu.RLock()
	n := len(r.state)
	r.mu.RUnlock()
	return n
}
