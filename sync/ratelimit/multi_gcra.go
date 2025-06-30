package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidMultiGCRALimit  = errors.New("multi GCRA limit must be positive")
	ErrInvalidMultiGCRAPeriod = errors.New("multi GCRA period must be positive")
	ErrInvalidMultiGCRABurst  = errors.New("multi GCRA burst cannot be negative")
)

type MultiGCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

	// Option.
	interval int64
	offset   int64
	period   int64
	Now      func() time.Time
}

func NewMultiGCRA(limit int, period time.Duration, burst int) (*MultiGCRA, error) {
	if limit <= 0 {
		return nil, ErrInvalidMultiGCRALimit
	}
	if period <= 0 {
		return nil, ErrInvalidMultiGCRAPeriod
	}
	if burst < 0 {
		return nil, ErrInvalidMultiGCRABurst
	}

	interval := period.Nanoseconds() / int64(limit)
	if interval == 0 {
		interval = 1
	}

	// Prevent integer overflow in burst calculations
	var offset int64
	if burst > 0 && interval > 0 {
		maxBurst := (1<<63 - 1) / interval
		if int64(burst) > maxBurst {
			offset = 1<<63 - 1
		} else {
			offset = interval * int64(burst)
		}
	}

	return &MultiGCRA{
		// NOTE: The burst is only applied once.
		state:    make(map[string]int64),
		interval: interval,
		offset:   offset,
		period:   period.Nanoseconds(),
		Now:      time.Now,
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
	if key == "" || n <= 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	r.state[key] = max(r.state[key], now)
	if r.state[key]-r.offset <= now {
		// Check for potential overflow before adding
		if r.state[key] > 0 && int64(n) > 0 && r.interval > 0 {
			maxAdd := (1<<63 - 1) - r.state[key]
			if int64(n)*r.interval > maxAdd {
				// Handle overflow by setting to max value
				r.state[key] = 1<<63 - 1
			} else {
				r.state[key] += int64(n) * r.interval
			}
		} else {
			r.state[key] += int64(n) * r.interval
		}

		return true
	}

	return false
}

func (r *MultiGCRA) RetryAt(key string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.state[key] < now.UnixNano() {
		return now
	}

	return time.Unix(0, r.state[key]+r.interval)
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
