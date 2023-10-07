package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	mu      sync.Mutex
	limit   int64
	period  int64
	resetAt int64
	count   int64
	Now     func() time.Time
}

func NewFixedWindow(limit int64, period time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}
}

func (rl *FixedWindow) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now().UnixNano()
	if rl.resetAt < now {
		rl.resetAt = now + rl.period
		rl.count = 0
	}

	resetIn := toNanosecond(rl.resetAt - now)

	var allow bool
	var remaining int64
	if rl.count+n <= rl.limit {
		rl.count += n
		allow = true
		remaining = rl.limit - rl.count
	}

	retryIn := resetIn
	if remaining > 0 {
		retryIn = 0
	}

	return &Result{
		Allow:     allow,
		Remaining: remaining,
		RetryIn:   retryIn,
		ResetIn:   resetIn,
	}
}

func (rl *FixedWindow) Allow() *Result {
	return rl.AllowN(1)
}

func toNanosecond(n int64) time.Duration {
	return time.Duration(n) * time.Nanosecond
}

type Result struct {
	Allow     bool
	Remaining int64
	RetryIn   time.Duration
	ResetIn   time.Duration
}
