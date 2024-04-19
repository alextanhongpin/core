package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu sync.Mutex

	// Option.
	burst  int64
	n      int64
	period time.Duration

	// State.
	interval time.Duration
	state    *state

	Now func() time.Time
}

type state struct {
	retryAt time.Time
	count   int64
	resetAt time.Time
}

func NewTokenBucket(n int64, period time.Duration, burst int64) *TokenBucket {
	return &TokenBucket{
		n:        n,
		period:   period,
		interval: period / time.Duration(n),
		burst:    burst,
		state:    new(state),
		Now:      time.Now,
	}
}

func (tb *TokenBucket) AllowAt(t time.Time, n int) bool {
	s := &state{
		resetAt: tb.state.resetAt,
		retryAt: tb.state.retryAt,
		count:   tb.state.count,
	}
	return tb.allowAtN(s, t, n).Allow
}

func (tb *TokenBucket) Allow() *Result {
	return tb.AllowN(1)
}

func (tb *TokenBucket) AllowN(n int) *Result {
	s := tb.state
	now := tb.Now()
	return tb.allowAtN(s, now, n)
}

func (tb *TokenBucket) allowAtN(s *state, now time.Time, n int) *Result {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.reset(s)

	var allow bool
	if !tb.allowFlow(s, n) {
		if tb.allowBurst(s, n) {
			allow = true
		}
	} else {
		allow = true
	}

	start := now.Truncate(tb.period)
	end := start.Add(tb.period)
	count := int64(end.Sub(s.retryAt)) / int64(tb.interval)

	res := new(Result)
	res.Limit = tb.burst + tb.n
	res.Remaining = max(tb.burst-s.count, 0) + count
	if s.count < tb.burst {
		res.RetryAt = now
	} else {
		res.RetryAt = s.retryAt
	}
	res.ResetAt = s.resetAt
	res.Allow = allow

	return res
}

// allowBurst returns true if the burst is allowed.
// Burst quota is not affected by the interval.
func (tb *TokenBucket) allowBurst(s *state, n int) bool {
	if s.count < tb.burst {
		s.count += int64(n)
		return true
	}

	return false
}

// allowFlow updates the checkpoint of the last request.
// If the checkpoint is the same as the current time, it returns false.
// Otherwise, it returns true and updates the checkpoint.
func (tb *TokenBucket) allowFlow(s *state, n int) bool {
	now := tb.Now()
	checkpoint := now.Truncate(tb.interval)
	if !checkpoint.Before(s.retryAt) {
		s.retryAt = checkpoint.Add(time.Duration(int(tb.interval) * n))
		return true
	}

	return false
}

func (tb *TokenBucket) reset(s *state) {
	now := tb.Now()

	if now.Before(s.resetAt) {
		return
	}

	start := now.Truncate(tb.period)
	end := start.Add(tb.period)
	s.retryAt = start
	s.resetAt = end
	s.count = 0
}
