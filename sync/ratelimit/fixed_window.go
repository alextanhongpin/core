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

// AllowAt allows performing a dry-run to check if the ratelimiter is allowed
// at the given time without consuming a token.
func (rl *FixedWindow) AllowAt(t time.Time, n int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limit := rl.limit
	count := rl.count

	if !rl.resetAt.After(t) {
		count = 0
	}

	return count+n <= limit
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (rl *FixedWindow) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now()
	// resetAt <= now
	if !rl.resetAt.After(now) {
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

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (rl *FixedWindow) Allow() *Result {
	return rl.AllowN(1)
}
