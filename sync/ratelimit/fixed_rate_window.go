package ratelimit

import (
	"context"
	"math"
	"time"
)

type FixedRateWindow struct {
	limit   int64
	every   time.Duration
	resetAt int64
	count   int64
	burst   int64
	Now     func() time.Time
}

func NewFixedRateWindow(limit int64, every time.Duration) *FixedRateWindow {
	return &FixedRateWindow{
		limit: limit,
		every: every,
		Now:   time.Now,
	}
}

func (rl *FixedRateWindow) inverse() int64 {
	return rl.every.Nanoseconds() / rl.limit
}

func (rl *FixedRateWindow) AllowN(ctx context.Context, key string, n int64) *Result {
	period := rl.every.Nanoseconds()
	now := rl.Now().UnixNano()

	windowStart := now - (now % period)
	windowEnd := windowStart + period

	if rl.resetAt < now {
		rl.resetAt = windowEnd
		rl.count = 0
	}

	quota := int64(math.Ceil(float64(now%period) / float64(period) * float64(rl.limit)))
	batch := now % period
	batchStart := batch - (batch % rl.inverse())
	batchEnd := batchStart + rl.inverse()
	retryIn := toNanosecond(batchEnd - batch)
	resetIn := toNanosecond(windowEnd - now)

	if rl.count+n <= quota+rl.burst {
		if rl.count+n <= rl.burst {
			retryIn = 0
		}

		rl.count += n

		return &Result{
			Allow:     true,
			Remaining: quota + rl.burst - rl.count,
			RetryIn:   retryIn,
			ResetIn:   resetIn,
		}
	}

	return &Result{
		RetryIn: retryIn,
		ResetIn: resetIn,
	}
}

func (rl *FixedRateWindow) Allow(ctx context.Context, key string) *Result {
	return rl.AllowN(ctx, key, 1)
}
