package ratelimit

import (
	"math"
	"sync"
	"time"
)

type SlidingWindow struct {
	// State.
	mu     sync.Mutex
	prev   int64
	curr   int64
	window int64

	// Options.
	limit  int64
	period int64

	Now func() time.Time
}

func NewSlidingWindow(limit int64, period time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:  limit,
		period: period.Nanoseconds(),
		Now:    time.Now,
	}
}

func (rl *SlidingWindow) Allow() *Result {
	return rl.AllowN(1)
}

func (rl *SlidingWindow) AllowN(n int64) *Result {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := int64(rl.Now().Nanosecond())
	curr := now / rl.period * rl.period
	prev := curr - rl.period

	if rl.window == prev {
		rl.prev = rl.curr
		rl.curr = 0
		rl.window = curr
	} else if rl.window != curr {
		rl.prev = 0
		rl.curr = 0
		rl.window = curr
	}

	ratio := float64(now-curr) / float64(rl.period)
	count := int64(math.Round((1-ratio)*float64(rl.prev) + float64(rl.curr)))

	res := &Result{
		Allow: count+n <= rl.limit,
		Limit: rl.limit,
	}

	if res.Allow {
		rl.curr += n

		res.Remaining = rl.limit - count - n
	}

	if res.Remaining == 0 {
		res.RetryAfter = time.Duration(now + rl.period)
	}

	return res
}
