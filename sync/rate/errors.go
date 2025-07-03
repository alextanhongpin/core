package rate

import (
	"sync"
	"time"
)

// Errors tracks success and failure rates over time using exponential decay.
// It combines two Rate counters to provide error rate calculations.
type Errors struct {
	mu      sync.Mutex
	success *Rate
	failure *Rate
}

// NewErrors creates a new error rate tracker with the specified time period.
// Both success and failure rates use the same period for consistency.
func NewErrors(period time.Duration) *Errors {
	return &Errors{
		success: NewRate(period),
		failure: NewRate(period),
	}
}

func (e *Errors) Reset() {
	e.mu.Lock()
	e.success.reset()
	e.failure.reset()
	e.mu.Unlock()
}

func (e *Errors) SetNow(now func() time.Time) {
	e.success.Now = now
	e.failure.Now = now
}

func (e *Errors) Success() counter {
	return e.success
}

func (e *Errors) Failure() counter {
	return e.failure
}

func (e *Errors) Rate() *ErrorRate {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Use the public Count() method to maintain proper encapsulation
	success := e.success.Count()
	failure := e.failure.Count()

	return &ErrorRate{
		success: success,
		failure: failure,
	}
}

// ErrorRate represents a snapshot of success and failure rates at a point in time.
type ErrorRate struct {
	failure float64
	success float64
}

// Success returns the success rate.
func (r *ErrorRate) Success() float64 {
	return r.success
}

// Failure returns the failure rate.
func (r *ErrorRate) Failure() float64 {
	return r.failure
}

// Total returns the total rate (success + failure).
func (r *ErrorRate) Total() float64 {
	return r.failure + r.success
}

// Ratio returns the failure ratio (failures / total).
// Returns 0 if there are no events.
func (r *ErrorRate) Ratio() float64 {
	num := r.failure
	den := r.failure + r.success
	if den <= 0 {
		return 0
	}

	return num / den
}
