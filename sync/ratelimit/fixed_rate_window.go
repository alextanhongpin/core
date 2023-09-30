package ratelimit

import (
	"math"
	"time"
)

type FixedRateWindow struct {
	limit   int64
	period  time.Duration
	burst   int64
	resetAt int64
	count   int64
	Now     func() time.Time
}

func NewFixedRateWindow(limit int64, period time.Duration, burst int64) *FixedRateWindow {
	return &FixedRateWindow{
		limit:  limit,
		period: period,
		burst:  burst,
		Now:    time.Now,
	}
}

func (rl *FixedRateWindow) inverse() int64 {
	return rl.period.Nanoseconds() / rl.limit
}

func (rl *FixedRateWindow) AllowN(n int64) *Result {
	period := rl.period.Nanoseconds()
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

		rl.count = max(rl.count, batch/rl.inverse())
		rl.count += n

		return &Result{
			Allow:     true,
			Remaining: max(quota+rl.burst-rl.count, 0),
			RetryIn:   retryIn,
			ResetIn:   resetIn,
		}
	}

	return &Result{
		RetryIn: resetIn,
		ResetIn: resetIn,
	}
}

func (rl *FixedRateWindow) Allow() *Result {
	return rl.AllowN(1)
}
