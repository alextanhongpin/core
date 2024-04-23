package rate

import (
	"sync"
	"time"
)

type ErrorCounter struct {
	mu      sync.RWMutex
	success *Counter
	failure *Counter
	Now     func() time.Time
}

func NewErrorCounter(window time.Duration) *ErrorCounter {
	return &ErrorCounter{
		success: NewCounter(window),
		failure: NewCounter(window),
		Now:     time.Now,
	}
}

func (e *ErrorCounter) MarkSuccess(n float64) float64 {
	e.mu.Lock()
	n = e.success.Inc(n)
	e.mu.Unlock()
	return n
}

func (e *ErrorCounter) MarkFailure(n float64) float64 {
	e.mu.Lock()
	n = e.failure.Inc(n)
	e.mu.Unlock()
	return n
}

func (e *ErrorCounter) Record(fn func() error) error {
	if err := fn(); err != nil {
		e.MarkFailure(1)
		return err
	}

	e.MarkSuccess(1)

	return nil
}

func (e *ErrorCounter) Success() float64 {
	e.mu.RLock()
	n := e.success.Get()
	e.mu.RUnlock()
	return n
}

func (e *ErrorCounter) Failure() float64 {
	e.mu.RLock()
	n := e.failure.Get()
	e.mu.RUnlock()
	return n
}

func (e *ErrorCounter) Rate() float64 {
	now := e.Now()

	e.mu.RLock()
	failure := e.failure.GetAt(now)
	success := e.success.GetAt(now)
	e.mu.RUnlock()

	if failure <= 0 {
		return failure
	}

	return failure / (failure + success)
}

type Counter struct {
	// Option.
	window time.Duration

	// State.
	count float64
	last  time.Time
	Now   func() time.Time
}

func NewCounter(window time.Duration) *Counter {
	return &Counter{
		window: window,
		Now:    time.Now,
	}
}

func (r *Counter) GetAt(t time.Time) float64 {
	return r.countWithDamping(t, r.last, r.window, r.count)
}

func (r *Counter) Get() float64 {
	return r.countWithDamping(r.Now(), r.last, r.window, r.count)
}

func (r *Counter) Inc(n float64) float64 {
	r.count = r.Get() + n
	r.last = r.Now()
	return r.count
}

func (r *Counter) countWithDamping(now time.Time, last time.Time, window time.Duration, count float64) float64 {
	elapsed := now.Sub(last)

	// This looks complicated, but is actually quite simple.
	// Say we have a time window of 1s. If the last call was made 0.5s, we can just drop half the requests.
	// This is what the calculation below is doing.
	// If the last call was made 0.1s ago, we can drop 10% of the request, and keep only 90% of it.
	// If the last call was made 0.9s ago, we can drop 90% of the request, and keep only 10% of it.
	rate := float64(elapsed) / float64(window)
	rate = min(1, rate)
	rate = 1 - rate
	count *= rate
	return count
}
