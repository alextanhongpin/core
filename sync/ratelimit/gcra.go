package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type GCRA struct {
	// State.
	mu   sync.RWMutex
	last int64

	// Option.
	Now       func() time.Time
	increment int64
	offset    int64
}

func NewGCRA(limit int, period time.Duration, burst int) (*GCRA, error) {
	o := &option{limit: limit, period: period, burst: burst}
	if err := o.Validate(); err != nil {
		return nil, err
	}

	increment := div(period.Nanoseconds(), int64(limit))
	if increment <= 0 {
		return nil, fmt.Errorf("%w: period divided by limit", ErrInvalidNumber)
	}
	offset := mul(increment, int64(burst))

	return &GCRA{
		// NOTE: The burst is only applied once.
		Now:       time.Now,
		increment: increment,
		offset:    offset,
	}, nil
}

// MustNewGCRA creates a new GCRA rate limiter and panics on error.
// This is provided for backward compatibility and testing.
func MustNewGCRA(limit int, period time.Duration, burst int) *GCRA {
	gcra, err := NewGCRA(limit, period, burst)
	if err != nil {
		panic(err)
	}
	return gcra
}

func (r *GCRA) Allow() bool {
	return r.AllowN(1)
}

func (r *GCRA) AllowN(n int) bool {
	if n <= 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.Now().UnixNano()
	r.last = max(r.last, now)
	if r.last-r.offset <= now {
		increment := mul(r.increment, int64(n))
		r.last = add(r.last, increment)
		return true
	}

	return false
}

func (r *GCRA) RetryAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	if r.last <= now.UnixNano() {
		return now
	}

	return time.Unix(0, r.last+r.increment)
}

func (r *GCRA) Remaining() int {
	return -1
}
