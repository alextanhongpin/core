package bench

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

type GCRA struct {
	// State.
	last *atomic.Int64

	// Option.
	offset    int64
	increment int64
	Now       func() time.Time
}

func NewGCRA(limit int, period time.Duration, burst int) (*GCRA, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit", ratelimit.ErrInvalidNumber)
	}
	if period <= 0 {
		return nil, fmt.Errorf("%w: period", ratelimit.ErrInvalidNumber)
	}
	if burst < 0 {
		return nil, fmt.Errorf("%w: burst", ratelimit.ErrInvalidNumber)
	}

	increment := period.Nanoseconds() / int64(limit)
	if increment <= 0 {
		return nil, fmt.Errorf("%w: period divided by limit", ratelimit.ErrInvalidNumber)
	}
	offset := increment * int64(burst)

	return &GCRA{
		last: new(atomic.Int64),
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

	now := r.Now().UnixNano()
	r.last.Store(max(r.last.Load(), now))
	if r.last.Load()-r.offset <= now {
		increment := r.increment * int64(n)
		r.last.Add(increment)
		return true
	}

	return false
}

func (r *GCRA) RetryAt() time.Time {
	now := r.Now()
	last := r.last.Load()
	if last <= now.UnixNano() {
		return now
	}

	return time.Unix(0, last+r.increment)
}
