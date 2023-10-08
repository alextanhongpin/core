package ratelimit

import (
	"sync"
	"time"
)

type LeakyBucket struct {
	mu sync.Mutex

	// Option.
	limit  int64
	period time.Duration
	burst  int64

	// State.
	resetAt  time.Time
	calledAt time.Time
	count    int64
	interval time.Duration
	batch    int64

	Now func() time.Time
}

func NewLeakyBucket(limit int64, period time.Duration, burst int64) *LeakyBucket {
	return &LeakyBucket{
		limit:    limit,
		period:   period,
		burst:    min(burst, limit),
		interval: period / time.Duration(limit),
		Now:      time.Now,
	}
}

func (rl *LeakyBucket) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now()
	if rl.resetAt.Before(now) {
		rl.resetAt = now.Add(rl.period)
		rl.count = 0
	}

	windowStart := rl.resetAt.Add(-rl.period)

	batchPeriod := now.Sub(windowStart)
	batch := int64(batchPeriod / rl.interval)
	batchStart := windowStart.Add(rl.interval * time.Duration(batch))
	batchEnd := batchStart.Add(rl.interval)

	resetAt := rl.resetAt

	// Allow burst.
	if rl.count+n <= rl.burst {
		rl.count += n

		return &Result{
			Allow:     true,
			Remaining: max(rl.limit-rl.count, 0),
			RetryAt:   now, // Can retry immediately.
			ResetAt:   resetAt,
		}
	}

	retryAt := resetAt
	var allow bool

	if batch+1 > rl.batch && rl.count+n <= rl.limit {
		// Expires count that is not used.
		rl.count = max(rl.count, batch)
		rl.count += n
		rl.batch = batch + 1

		allow = true
	}

	remaining := max(rl.limit-rl.count, 0)
	if remaining > 0 {
		// If no more tokens remaining, we can only try at the next reset at.
		retryAt = batchEnd
	}

	return &Result{
		Allow:     allow,
		Remaining: remaining,
		RetryAt:   retryAt,
		ResetAt:   resetAt,
	}
}

func (rl *LeakyBucket) Allow() *Result {
	return rl.AllowN(1)
}
