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
	return &successTx{e: e}
}

func (e *Errors) Failure() counter {
	return &failureTx{e: e}
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

type successTx struct {
	e *Errors
}

func (r *successTx) Add(f float64) float64 {
	r.e.mu.Lock()
	success := r.e.success.add(f)
	_ = r.e.failure.add(0)
	r.e.mu.Unlock()

	return success
}

func (r *successTx) Inc() float64 {
	return r.Add(1)
}

func (r *successTx) Count() float64 {
	return r.Add(0)
}

type failureTx struct {
	e *Errors
}

func (r *failureTx) Add(f float64) float64 {
	r.e.mu.Lock()
	failure := r.e.failure.add(f)
	_ = r.e.success.add(0)
	r.e.mu.Unlock()

	return failure
}

func (r *failureTx) Inc() float64 {
	return r.Add(1)
}

func (r *failureTx) Count() float64 {
	return r.Add(0)
}
