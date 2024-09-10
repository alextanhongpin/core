package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	// State.
	mu      sync.RWMutex
	count   int64
	resetAt int64

	// Options.
	limit  int64
	window int64
	Now    func() time.Time
}

func NewFixedWindow(limit int64, period time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:  limit,
		window: period.Nanoseconds(),
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

	if rl.resetAt <= t.UnixNano() {
		count = 0
	}

	return count+n <= limit
}

type FixedWindowResult struct {
	Allow      bool
	RetryAfter time.Duration
	Remaining  int64
}

// AllowN checks if a request is allowed. Consumes n token
// if allowed.
func (rl *FixedWindow) AllowN(n int64) FixedWindowResult {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now().UnixNano()
	if rl.resetAt <= now {
		rl.count = 0
		rl.resetAt = now + rl.window
	}

	var res FixedWindowResult
	allow := rl.count+n <= rl.limit
	if allow {
		rl.count += n

		res.Allow = true
		res.Remaining = rl.limit - rl.count
	}

	if res.Remaining == 0 {
		res.RetryAfter = time.Duration(rl.resetAt - now)
	}

	return res
}

// Allow checks if a request is allowed. Special case of AllowN that consumes
// only 1 token.
func (rl *FixedWindow) Allow() FixedWindowResult {
	return rl.AllowN(1)
}
