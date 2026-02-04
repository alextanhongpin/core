package ratelimit

import (
	"math"
	"time"
)

// SlidingWindow implements a sliding window rate limiter.
type SlidingWindow struct {
	// State.
	prev   int
	curr   int
	window int64

	// Options.
	limit  int
	period int64

	Now func() time.Time
}

func NewSlidingWindow(limit int, period time.Duration) (*SlidingWindow, error) {
	o := &option{limit: limit, period: period}
	if err := o.Validate(); err != nil {
		return nil, err
	}

	return &SlidingWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}, nil
}

// MustNewSlidingWindow creates a new sliding window rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewSlidingWindow(limit int, period time.Duration) *SlidingWindow {
	sw, err := NewSlidingWindow(limit, period)
	if err != nil {
		panic(err)
	}
	return sw
}

func (r *SlidingWindow) Allow() bool {
	return r.AllowN(1)
}

func (r *SlidingWindow) AllowN(n int) bool {
	if n <= 0 {
		return false
	}

	if r.remaining() >= n {
		r.add(n)
		return true
	}

	return false
}

func (r *SlidingWindow) RetryAt() time.Time {
	panic("not implemented")
}

func (r *SlidingWindow) Remaining() int {
	return r.remaining()
}

func (r *SlidingWindow) remaining() int {
	now := r.Now().UnixNano()

	// [t0 + dt][t1 + dt]
	// ....t1... (now < t0 + dt)
	// ...............t1 (now < t0 * 2*dt)
	prev := r.prev
	curr := r.curr
	window := r.window

	if window+r.period > now {
		// In current window
	} else if window+2*r.period > now {
		// In previous window
		prev = r.curr
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

func (r *SlidingWindow) add(n int) {
	now := r.Now().UnixNano()
	if r.window+r.period > now {
		// In current window
	} else if r.window+2*r.period > now {
		// In previous window
		r.prev = r.curr
		r.curr = 0
		r.window += r.period
	} else {
		r.prev = 0
		r.curr = 0
		r.window = now
	}

	r.curr += n
}
