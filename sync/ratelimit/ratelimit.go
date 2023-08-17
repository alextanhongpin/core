package ratelimit

import (
	"time"
)

type Limiter struct {
	name     string
	count    int           // The current number of request.
	limit    int           // The number of request.
	interval time.Duration // The period for the request limit.
	last     time.Time     // The last time it was executed.
	now      func() time.Time
}

func New(name string, n int, interval time.Duration) *Limiter {
	return &Limiter{
		name:     name,
		limit:    n,
		interval: interval,
		now:      time.Now,
	}
}

// period returns the time taken for one request to complete.
func (l *Limiter) Period() time.Duration {
	return l.interval / time.Duration(l.limit)
}

func (l *Limiter) Remaining() int {
	return max(l.limit-l.count, 0)
}

func (l *Limiter) Limit() int {
	return l.limit
}

func (l *Limiter) Allow() bool {
	period := l.Period()
	now := l.now()

	end := l.last.Add(l.interval)
	if end.Before(now) {
		// Reset.
		l.last = now.Truncate(period)
		l.count = 0
	}

	prev := l.last.Add(period * time.Duration(l.count))
	next := prev.Add(period)

	if !between(now, prev, next) {
		return false
	}

	l.count++

	return l.count <= l.limit
}

func (l *Limiter) SetNow(now func() time.Time) {
	l.now = now
}

// between returns true if time t fulfils: min <= t < max
func between(a, lo, hi time.Time) bool {
	return !a.Before(lo) && a.Before(hi)
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
