package ratelimit

import (
	"context"
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	limit   int64
	every   time.Duration
	resetAt int64
	count   int64
	Now     func() time.Time
}

func NewFixedWindow(limit int64, every time.Duration) *FixedWindow {
	return &FixedWindow{
		limit: limit,
		every: every,
		Now:   time.Now,
	}
}

func (rl *FixedWindow) AllowN(ctx context.Context, key string, n int64) *Result {
	period := rl.every.Nanoseconds()
	now := rl.Now().UnixNano()
	if rl.resetAt < now {
		rl.resetAt = now - (now % period) + period
		rl.count = 0
	}

	t := toNanosecond(period - now%period)

	if rl.count+n <= rl.limit {
		rl.count += n

		return &Result{
			Allow:     true,
			Remaining: rl.limit - rl.count,
			RetryIn:   0,
			ResetIn:   t,
		}
	}

	return &Result{
		RetryIn: t,
		ResetIn: t,
	}
}

func (rl *FixedWindow) Allow(ctx context.Context, key string) *Result {
	return rl.AllowN(ctx, key, 1)
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
