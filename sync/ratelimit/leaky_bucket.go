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

func (rl *LeakyBucket) AllowAt(t time.Time, n int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	resetAt := rl.resetAt
	count := rl.count
	batch := rl.batch

	if !rl.resetAt.After(t) {
		resetAt = t.Add(rl.period)
		count = 0
		batch = 0
	}

	windowStart := resetAt.Add(-rl.period)

	batchPeriod := t.Sub(windowStart)
	newBatch := int64(batchPeriod / rl.interval)

	if rl.count+n <= rl.burst {
		return true
	}

	return newBatch+1 > batch && count+n <= rl.limit
}

func (rl *LeakyBucket) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.Now()
	// resetAt <= now
	if !rl.resetAt.After(now) {
		rl.resetAt = now.Add(rl.period)
		rl.count = 0
		rl.batch = 0
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
	allow := batch+1 > rl.batch && rl.count+n <= rl.limit
	if allow {
		// Expires count that is not used.
		rl.count = max(rl.count, batch)
		rl.count += n
		rl.batch = batch + 1
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
