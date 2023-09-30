package ratelimit

import "time"

// GCRA implements the Genetic-Cell-Rate-Algorithm.
type GCRA struct {
	limit   int64
	period  time.Duration
	burst   int64
	resetAt int64

	Now func() time.Time
}

func NewGCRA(limit int64, period time.Duration, burst int64) *GCRA {
	return &GCRA{
		limit:  limit,
		period: period,
		burst:  burst,
		Now:    time.Now,
	}
}

func (g *GCRA) Allow() *Result {
	return g.AllowN(1)
}

func (g *GCRA) AllowN(n int64) *Result {
	now := g.Now().UnixNano()
	period := g.period.Nanoseconds()

	windowStart := now - (now % period)
	windowEnd := windowStart + period

	batch := now - windowStart
	batchStart := batch - (batch % g.interval())
	batchEnd := batchStart + g.interval()

	if g.resetAt < now {
		g.resetAt = windowStart + batchStart
	}

	allowAt := g.resetAt - g.interval()*(g.burst)
	allow := now >= allowAt
	if allow {
		g.resetAt += g.interval() * n
	}

	retryIn := toNanosecond(batchEnd - batch)
	resetIn := toNanosecond(windowEnd - now)

	remaining := max(g.limit-(g.resetAt-windowStart)/g.interval()+g.burst, 0)
	if g.burst > 0 && remaining > g.burst {
		retryIn = 0
	}

	return &Result{
		Allow:     allow,
		Remaining: remaining,
		RetryIn:   retryIn,
		ResetIn:   resetIn,
	}
}

func (g *GCRA) interval() int64 {
	return g.period.Nanoseconds() / g.limit
}
