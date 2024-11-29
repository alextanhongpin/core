package rate

import (
	"sync"
	"time"
)

type Errors struct {
	mu      sync.Mutex
	Success *Rate
	Failure *Rate
}

func NewErrors(period time.Duration) *Errors {
	return &Errors{
		Success: NewRate(period),
		Failure: NewRate(period),
	}
}

func (e *Errors) Reset() {
	e.mu.Lock()
	e.Success.reset()
	e.Failure.reset()
	e.mu.Unlock()
}

func (e *Errors) SetNow(now func() time.Time) {
	e.Success.Now = now
	e.Failure.Now = now
}

func (e *Errors) IncSuccess() *Result {
	return e.AddSuccess(1)
}

func (e *Errors) AddSuccess(f float64) *Result {
	e.mu.Lock()
	success := e.Success.add(f)
	failure := e.Failure.add(0)
	e.mu.Unlock()

	return &Result{
		Success: success,
		Failure: failure,
	}
}

func (e *Errors) IncFailure() *Result {
	return e.AddFailure(1)
}

func (e *Errors) AddFailure(f float64) *Result {
	e.mu.Lock()
	success := e.Success.add(0)
	failure := e.Failure.add(f)
	e.mu.Unlock()

	return &Result{
		Success: success,
		Failure: failure,
	}
}

func (e *Errors) Result() *Result {
	e.mu.Lock()
	success := e.Success.add(0)
	failure := e.Failure.add(0)
	e.mu.Unlock()

	return &Result{
		Success: success,
		Failure: failure,
	}
}

type Result struct {
	Failure float64
	Success float64
}

func (r *Result) Total() float64 {
	return r.Failure + r.Success
}

func (r *Result) ErrorRate() float64 {
	num := r.Failure
	den := r.Failure + r.Success
	if den <= 0 {
		return 0
	}

	return num / den
}
