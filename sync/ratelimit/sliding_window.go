package ratelimit

import (
	"errors"
	"math"
	"sync"
	"time"
)

var (
	ErrInvalidSlidingWindowLimit  = errors.New("sliding window limit must be positive")
	ErrInvalidSlidingWindowPeriod = errors.New("sliding window period must be positive")
)

// SlidingWindow implements a sliding window rate limiter.
type SlidingWindow struct {
	// State.
	mu     sync.RWMutex
	prev   int
	curr   int
	window int64

	// Options.
	limit  int
	period int64

	Now func() time.Time

	metricsCollector MetricsCollector
}

func NewSlidingWindow(limit int, period time.Duration) (*SlidingWindow, error) {
	if limit <= 0 {
		return nil, ErrInvalidSlidingWindowLimit
	}
	if period <= 0 {
		return nil, ErrInvalidSlidingWindowPeriod
	}

	return &SlidingWindow{
		limit:            limit,
		period:           period.Nanoseconds(),
		Now:              time.Now,
		metricsCollector: &AtomicMetricsCollector{},
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

func (r *SlidingWindow) WithMetricsCollector(collector MetricsCollector) *SlidingWindow {
	if collector != nil {
		r.metricsCollector = collector
	}
	return r
}

func (r *SlidingWindow) Allow() bool {
	r.metricsCollector.IncTotalRequests()
	allowed := r.AllowN(1)
	if allowed {
		r.metricsCollector.IncAllowed()
	} else {
		r.metricsCollector.IncDenied()
	}
	return allowed
}

func (r *SlidingWindow) AllowN(n int) bool {
	r.metricsCollector.IncTotalRequests()
	if n <= 0 {
		r.metricsCollector.IncDenied()
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.remaining() >= n {
		r.add(n)
		return true
	}

	return false
}

func (r *SlidingWindow) Remaining() int {
	r.mu.RLock()
	n := r.remaining()
	r.mu.RUnlock()

	return n
}

func (r *SlidingWindow) remaining() int {
	now := r.Now().UnixNano()

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
