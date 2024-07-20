package rate

import (
	"math"
	"sync"
	"time"
)

func NewRate(period time.Duration) *Rate {
	return &Rate{
		period: period,
		Now:    time.Now,
	}
}

type Rate struct {
	mu     sync.Mutex
	count  float64
	last   time.Time
	period time.Duration
	Now    func() time.Time
}

func (r *Rate) Inc(n int64) int64 {
	r.mu.Lock()
	f := r.inc(n)
	r.mu.Unlock()
	return int64(math.Ceil(f))
}

func (r *Rate) inc(n int64) float64 {
	f := dampFactor(r.Now().Sub(r.last), r.period)
	r.count *= f
	r.count += float64(n)
	r.last = time.Now()
	return r.count
}

func dampFactor(d time.Duration, period time.Duration) float64 {
	// It's in the future.
	if d < 0 {
		d = 0
	}
	// It's way in the past.
	if d > period {
		d = period
	}
	return float64(period-d) / float64(period)
}
