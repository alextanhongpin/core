package ratelimit

import (
	"math"
	"time"
)

type GCRA struct {
	// State.
	ts time.Time

	// Option.
	burst  int64
	limit  int64
	period time.Duration
}

func NewGCRA(limit int64, period time.Duration, burst int64) *GCRA {
	return &GCRA{
		limit:  limit,
		period: period,
		burst:  burst,
	}
}

func (g *GCRA) Allow() *Result {
	return g.AllowN(1)
}

func (g *GCRA) AllowN(n int) *Result {
	now := time.Now()
	interval := g.period / time.Duration(g.limit)
	burst := time.Duration(g.burst) * interval
	token := time.Duration(n) * interval

	if lt(g.ts, now) {
		g.ts = now
	}

	allow := false
	if lte(g.ts.Add(-burst), now) {
		allow = true
		g.ts = g.ts.Add(token)
	}

	resetAt := now.Truncate(g.period).Add(g.period)
	remaining := int64(math.Floor(float64(resetAt.Sub(now)) / float64(interval)))

	return &Result{
		Allow:     allow,
		Limit:     g.limit + g.burst,
		Remaining: remaining,
		ResetAt:   resetAt,
		RetryAt:   g.ts,
	}
}

func lt(a, b time.Time) bool {
	return a.Before(b)
}

func lte(a, b time.Time) bool {
	return !a.After(b)
}
