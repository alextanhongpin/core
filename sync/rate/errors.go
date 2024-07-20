package rate

import (
	"sync"
	"time"
)

type Errors struct {
	mu      sync.Mutex
	success *Rate
	failure *Rate
}

func NewErrors(period time.Duration) *Errors {
	return &Errors{
		success: NewRate(period),
		failure: NewRate(period),
	}
}

func (e *Errors) Inc(n int64) float64 {
	var s, f float64
	switch {
	case n < 0:
		e.mu.Lock()
		f = e.failure.inc(-n)
		s = e.success.inc(0)
		e.mu.Unlock()
	case n > 0:
		e.mu.Lock()
		s = e.success.inc(n)
		f = e.failure.inc(0)
		e.mu.Unlock()
	}

	num := f
	den := f + s
	if den == 0.0 {
		return 0.0
	}
	return num / den
}

func (e *Errors) SetNow(now time.Time) {
	f := func() time.Time {
		return now
	}

	e.success.Now = f
	e.failure.Now = f
}
