package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type MultiGCRA struct {
	// State.
	mu    sync.RWMutex
	state map[string]int64

	// Option.
	Now       func() time.Time
	increment int64
	offset    int64
	period    int64
}

func NewMultiGCRA(limit int, period time.Duration, burst int) (*MultiGCRA, error) {
	o := &option{limit: limit, period: period, burst: burst}
	if err := o.Validate(); err != nil {
		return nil, err
	}

	increment := div(period.Nanoseconds(), int64(limit))
	if increment == 0 {
		return nil, fmt.Errorf("%w: period divided by limit", ErrInvalidNumber)
	}
	offset := mul(increment, int64(burst))
	return &MultiGCRA{
		// NOTE: The burst is only applied once.
		increment: increment,
		offset:    offset,
		period:    period.Nanoseconds(),
		state:     make(map[string]int64),
		Now:       time.Now,
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
		increment := mul(r.increment, int64(n))
		r.state[key] = add(r.state[key], increment)
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

	return time.Unix(0, r.state[key]+r.increment)
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
