package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidLimit  = errors.New("limit must be positive")
	ErrInvalidPeriod = errors.New("period must be positive")
	ErrInvalidBurst  = errors.New("burst cannot be negative")
	ErrInvalidN      = errors.New("n must be positive")
)

type GCRA struct {
	// State.
	mu   sync.RWMutex
	last int64

	// Option.
	offset   int64
	interval int64
	Now      func() time.Time
}

func NewGCRA(limit int, period time.Duration, burst int) (*GCRA, error) {
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	if period <= 0 {
		return nil, ErrInvalidPeriod
	}
	if burst < 0 {
		return nil, ErrInvalidBurst
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

	return &GCRA{
		// NOTE: The burst is only applied once.
		offset:   offset,
		interval: interval,
		Now:      time.Now,
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
		// Check for potential overflow before adding
		if r.last > 0 && int64(n) > 0 && r.interval > 0 {
			maxAdd := (1<<63 - 1) - r.last
			if int64(n)*r.interval > maxAdd {
				// Handle overflow by setting to max value
				r.last = 1<<63 - 1
			} else {
				r.last += int64(n) * r.interval
			}
		} else {
			r.last += int64(n) * r.interval
		}

		return true
	}

	return false
}

func (r *GCRA) RetryAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := r.Now()
	nowNano := now.UnixNano()

	if r.last <= nowNano {
		return now
	}

	// Check for potential overflow when adding interval
	if r.last > 0 && r.interval > 0 && r.last > (1<<63-1)-r.interval {
		// Handle overflow by returning a far future time
		return time.Unix(0, 1<<63-1)
	}

	return time.Unix(0, r.last+r.interval)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
