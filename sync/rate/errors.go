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
	success := e.success.add(0)
	failure := e.failure.add(0)
	e.mu.Unlock()

	return &ErrorRate{
		success: success,
		failure: failure,
	}
}

type ErrorRate struct {
	failure float64
	success float64
}

func (r *ErrorRate) Success() float64 {
	return r.success
}

func (r *ErrorRate) Failure() float64 {
	return r.failure
}

func (r *ErrorRate) Total() float64 {
	return r.failure + r.success
}

func (r *ErrorRate) Ratio() float64 {
	num := r.failure
	den := r.failure + r.success
	if den <= 0 {
		return 0
	}

	return num / den
}
