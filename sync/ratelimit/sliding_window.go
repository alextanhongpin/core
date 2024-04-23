package ratelimit

import (
	"math"
	"sync"
	"time"
)

type SlidingWindow struct {
	mu      sync.Mutex
	windows map[int64]int64
	limit   int64
	period  time.Duration
	Now     func() time.Time
}

func NewSlidingWindow(limit int64, period time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:   limit,
		period:  period,
		Now:     time.Now,
		windows: make(map[int64]int64),
	}
}

func (rl *SlidingWindow) Allow() *Result {
	return rl.AllowN(1)
}

func (rl *SlidingWindow) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tnow := rl.Now()

	tcurr := tnow.Truncate(rl.period)
	tprev := tcurr.Add(-rl.period)
	tnext := tcurr.Add(rl.period)

	// Clear old keys.
	if len(rl.windows) > 2 {
		for key := range rl.windows {
			if key < tprev.Unix() {
				delete(rl.windows, key)
			}
		}
	}

	prev := rl.windows[tprev.Unix()]
	curr := rl.windows[tcurr.Unix()]

	ratio := float64(tnow.Sub(tcurr)) / float64(rl.period)

	ratio = 1 - ratio
	count := ratio*float64(prev) + float64(curr)

	c := int64(math.Round(count))
	if c+n <= rl.limit {
		rl.windows[tcurr.Unix()] += n

		return &Result{
			Allow:     true,
			Limit:     rl.limit,
			Remaining: rl.limit - c - n,
			RetryAt:   rl.Now(),
			ResetAt:   tnext,
		}
	}

	num := rl.limit - n - curr
	den := prev
	rat := float64(num) / float64(den)
	rat = max(0, rat)
	rat = min(1, rat)
	rat = 1 - rat
	sleep := time.Duration(rat*float64(rl.period)) - tnow.Sub(tcurr)
	if num <= 0 {
		sleep = rl.period
	}
	retryAt := tnow.Add(sleep)

	return &Result{
		Allow:     false,
		Limit:     rl.limit,
		Remaining: 0,
		RetryAt:   retryAt,
		ResetAt:   tnext,
	}
}
