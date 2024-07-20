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

func (e *Errors) Inc(n int64) (sucesses, failures float64) {
	var s, f int64
	switch {
	case n < 0:
		f = -n
	case n > 0:
		s = n
	}

	e.mu.Lock()
	failures = e.failure.inc(f)
	sucesses = e.success.inc(s)
	e.mu.Unlock()

	return
}

func (e *Errors) SetNow(now time.Time) {
	f := func() time.Time {
		return now
	}

	e.success.Now = f
	e.failure.Now = f
}
