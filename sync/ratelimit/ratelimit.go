package ratelimit

import (
	"sort"
	"time"
)

type Limiter struct {
	count    int           // The current number of request.
	limit    int           // The number of request.
	interval time.Duration // The period for the request limit.
	last     time.Time     // The last time it was executed.
	now      func() time.Time
}

func New(n int, interval time.Duration) *Limiter {
	return &Limiter{
		limit:    n,
		interval: interval,
		now:      time.Now,
	}
}

func (l *Limiter) Remaining() int {
	return max(l.limit-l.count, 0)
}

func (l *Limiter) Limit() int {
	return l.limit
}

func (l *Limiter) AllowN(n int) bool {
	now := l.now()

	end := l.last.Add(l.interval)
	if end.Before(now) {
		// Reset.
		l.last = now
		l.count = 0
	}

	l.count += n

	return l.count <= l.limit
}

func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}

func (l *Limiter) SetNow(now func() time.Time) {
	l.now = now
}

type MultiRateLimiter struct {
	limiters []*Limiter
}

type MultiOption struct {
	Month  int
	Day    int
	Hour   int
	Minute int
	Second int
}

func (m *MultiOption) ToLimiters() []*Limiter {
	var res []*Limiter

	if m.Month > 0 {
		res = append(res, New(m.Month, 30*24*time.Hour))
	}

	if m.Day > 0 {
		res = append(res, New(m.Day, 24*time.Hour))
	}

	if m.Hour > 0 {
		res = append(res, New(m.Hour, time.Hour))
	}

	if m.Minute > 0 {
		res = append(res, New(m.Minute, time.Minute))
	}

	if m.Second > 0 {
		res = append(res, New(m.Second, time.Second))
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].interval < res[j].interval
	})

	return res
}

func NewMulti(opt MultiOption) *MultiRateLimiter {
	return &MultiRateLimiter{
		limiters: opt.ToLimiters(),
	}
}

func (r *MultiRateLimiter) SetNow(now func() time.Time) {
	for _, lim := range r.limiters {
		lim.SetNow(now)
	}
}

func (r *MultiRateLimiter) Remaining() int {
	lim := r.limiters[len(r.limiters)-1]
	return lim.Remaining()
}

func (r *MultiRateLimiter) Allow() bool {
	for _, lim := range r.limiters {
		if !lim.Allow() {
			return false
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
