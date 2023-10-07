package ratelimit

import (
	"sort"
	"time"
)

type MultiFixedWindow struct {
	limiters []*FixedWindow
}

type Rate struct {
	Limit  int64
	Period time.Duration
}

func (r *Rate) Interval() time.Duration {
	return r.Period / time.Duration(r.Limit)
}

func NewMultiFixedWindow(rates ...Rate) *MultiFixedWindow {
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Interval() < rates[j].Interval()
	})

	limiters := make([]*FixedWindow, len(rates))
	for i, r := range rates {
		limiters[i] = NewFixedWindow(r.Limit, r.Period)
	}
	return &MultiFixedWindow{
		limiters: limiters,
	}
}

func (rl *MultiFixedWindow) AllowN(n int64) *Result {
	result := make([]*Result, len(rl.limiters))

	for i, lim := range rl.limiters {
		result[i] = lim.AllowN(n)
		if !result[i].Allow {
			return result[i]
		}
	}

	return result[len(result)-1]
}

func (rl *MultiFixedWindow) Allow() *Result {
	return rl.AllowN(1)
}
