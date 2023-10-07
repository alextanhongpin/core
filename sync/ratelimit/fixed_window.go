package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	mu      sync.Mutex
	limit   int64
	period  time.Duration
	resetAt time.Time
	count   int64
	Now     func() time.Time
}

func NewFixedWindow(limit int64, period time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:  limit,
		period: period,
		Now:    time.Now,
	}
}

func (rl *FixedWindow) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now()
	if rl.resetAt.Before(now) {
		rl.resetAt = now.Add(rl.period)
		rl.count = 0
	}

	allow := rl.count+n <= rl.limit
	if allow {
		rl.count += n
	}

	remaining := max(rl.limit-rl.count, 0)
	resetAt := rl.resetAt
	retryAt := resetAt

	if remaining > 0 {
		retryAt = now
	}

	return &Result{
		Allow:     allow,
		Limit:     rl.limit,
		Remaining: remaining,
		RetryAt:   retryAt,
		ResetAt:   resetAt,
	}
}

func (rl *FixedWindow) Allow() *Result {
	return rl.AllowN(1)
}
