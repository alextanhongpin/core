// Package circuitbreaker is an in-memory implementation of circuit breaker.
// The idea is that each local node (server) should maintain it's own knowledge
// of the service availability, instead of depending on external infrastructure
// like distributed cache.

package circuitbreaker

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

const (
	timeout   = 5 * time.Second
	success   = 5                // min 5 success before the circuit breaker becomes closed.
	failure   = 10               // min 10 failures before the circuit breaker becomes open.
	errRate   = 0.9              // at least 90% of the requests fails.
	errPeriod = 10 * time.Second // time window to measure the error rate.
)

// errHandler checks if the error will cause the circuitbreaker to trip.
var errHandler = func(err error) bool {
	return err != nil
}

// Unavailable returns the error when the circuit breaker is not available.
var Unavailable = errors.New("circuit-breaker: unavailable")

// CircuitBreaker represents the circuit breaker.
type CircuitBreaker struct {
	// State.
	status      int64
	counter     int64
	total       int64
	deadline    int64 // deadline in nanosecond
	errDeadline int64 // deadline in nanosecond

	// Options.
	success    int64
	failure    int64
	timeout    time.Duration
	now        func() time.Time
	errHandler func(error) bool
	errRate    float64
	errPeriod  time.Duration
}

type Option struct {
	Success      int64
	Failure      int64
	Timeout      time.Duration
	Now          func() time.Time
	ErrorHandler func(err error) bool
	ErrorRate    float64
	ErrorPeriod  time.Duration
}

func NewOption() *Option {
	return &Option{
		Timeout:      timeout,
		Success:      success,
		Failure:      failure,
		Now:          time.Now,
		ErrorHandler: errHandler,
		ErrorRate:    errRate,
		ErrorPeriod:  errPeriod,
	}
}

// New returns a pointer to CircuitBreaker.
func New(opt *Option) *CircuitBreaker {
	if opt == nil {
		opt = NewOption()
	}

	if opt.Now == nil {
		opt.Now = time.Now
	}

	return &CircuitBreaker{
		timeout:    opt.Timeout,
		success:    opt.Success,
		failure:    opt.Failure,
		now:        opt.Now,
		errHandler: opt.ErrorHandler,
		errRate:    opt.ErrorRate,
		errPeriod:  opt.ErrorPeriod,
	}
}

// Exec updates the circuit breaker status based on the returned error.
func (cb *CircuitBreaker) Exec(ctx context.Context, h func(ctx context.Context) error) error {
	if !cb.allow(ctx) {
		return Unavailable
	}

	err := h(ctx)
	isErr := cb.errHandler(err)
	cb.update(!isErr)

	return err
}

// ResetIn returns the wait time before the service can be called again.
func (cb *CircuitBreaker) ResetIn() time.Duration {
	if !cb.Status().IsOpen() {
		return 0
	}

	t := fromUnixNano(cb.deadline)
	delta := t.Sub(cb.now())
	if delta < 0 {
		return 0
	}

	return delta
}

func (cb *CircuitBreaker) Status() Status {
	return Status(atomic.LoadInt64(&cb.status))
}

func (cb *CircuitBreaker) allow(ctx context.Context) bool {
	if cb.Status().IsOpen() {
		cb.update(true)
	}

	return !cb.Status().IsOpen()
}

func (cb *CircuitBreaker) update(ok bool) {
	switch cb.Status() {
	case Open:
		if cb.isTimerExpired() {
			cb.reset()
			cb.transition(Open, HalfOpen)
		}
	case HalfOpen:
		// The service is still unhealthy
		// Reset the counter and revert to Open.
		if !ok {
			cb.reset()
			cb.transition(HalfOpen, Open)

			return
		}

		// The service is healthy.
		// After a certain threshold, circuit breaker becomes Closed.
		if cb.incr(&cb.counter) > cb.success {
			cb.reset()
			cb.transition(HalfOpen, Closed)
		}
	case Closed:
		// Increment total requests.
		total := cb.incr(&cb.total)

		// The service is healthy. Do nothing.
		if ok {
			return
		}

		// Increment failed requests.
		// This is necessary to measure error rate.
		failed := cb.incr(&cb.counter)
		if failed == 1 {
			// Start recording at the first failure.
			cb.startWindow()
			return
		}

		if cb.isWindowExpired() {
			cb.reset()
			cb.update(ok)
			return
		}

		// The service is unhealthy. After a certain threshold, circuit breaker
		// becomes Open.
		if cb.isTripped(failed, total) {
			cb.startTimer()
			cb.transition(Closed, Open)
		}
	}
}

func (cb *CircuitBreaker) reset() {
	atomic.StoreInt64(&cb.total, 0)
	atomic.StoreInt64(&cb.counter, 0)
	atomic.StoreInt64(&cb.deadline, 0)
	atomic.StoreInt64(&cb.errDeadline, 0)
}

func (cb *CircuitBreaker) isTripped(failed, total int64) bool {
	if failed <= cb.failure {
		return false
	}

	// Just checking the threshold above is insufficient, because we can have
	// more successful requests than failed one.
	// For example, if our error threshold is 10 errors, but we have 90
	// successful requests but just 10 failed requests, the circuitbreaker should
	// not trip.
	// Instead, we calculate the error rate given this formula:
	//
	// error rate = failed requests / total requests
	//
	// Given the example below, and when we set the error rate threshold to 0.9,
	//
	// error rate = 10 / (10 + 90) = 0.1
	//
	// The circuitbreaker should not trip because we did not hit the error rate
	// threshold.
	if rate(failed, total) < cb.errRate {
		return false
	}

	return true
}

// startTimer sets the deadline when the circuit breaker allows requests to go through again.
func (cb *CircuitBreaker) startTimer() {
	atomic.StoreInt64(&cb.deadline, toUnixNano(cb.now().Add(cb.timeout)))
}

// startWindow sets a new error window to calculate the error rate.
func (cb *CircuitBreaker) startWindow() {
	atomic.StoreInt64(&cb.errDeadline, toUnixNano(cb.now().Add(cb.errPeriod)))
}

func (cb *CircuitBreaker) isTimerExpired() bool {
	return cb.isExpired(&cb.deadline)
}

func (cb *CircuitBreaker) isWindowExpired() bool {
	return cb.isExpired(&cb.errDeadline)
}

func (cb *CircuitBreaker) incr(n *int64) int64 {
	return atomic.AddInt64(n, 1)
}

func (cb *CircuitBreaker) isExpired(ns *int64) bool {
	t := fromUnixNano(atomic.LoadInt64(ns))
	return !t.IsZero() && cb.now().After(t)
}

func (cb *CircuitBreaker) transition(from, to Status) {
	atomic.CompareAndSwapInt64(&cb.status, from.Int64(), to.Int64())
}

func rate(count, total int64) float64 {
	return float64(count) / float64(total)
}

func toUnixNano(t time.Time) int64 {
	return t.UnixNano()
}

func fromUnixNano(ns int64) time.Time {
	return time.Unix(ns/1e9, ns%1e9)
}
