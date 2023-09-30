package ratelimit

import (
	"time"
)

// FixedWindow acts as a counter for a given time period.
type FixedWindow struct {
	limit   int64
	period  time.Duration
	resetAt int64
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
	period := rl.period.Nanoseconds()
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
