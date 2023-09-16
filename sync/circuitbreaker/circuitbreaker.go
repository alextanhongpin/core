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

	"golang.org/x/exp/event"
)

var (
	requestsTotal = event.NewCounter("requests_total", &event.MetricOptions{
		Description: "The number of executions",
	})

	failuresTotal = event.NewCounter("failures_total", &event.MetricOptions{
		Description: "The number of failures",
	})
)

const (
	timeout = 5 * time.Second
	success = 5  // 5 success before the circuit breaker becomes closed.
	failure = 10 // 10 failures before the circuit breaker becomes open.
)

// Unavailable returns the error when the circuit breaker is not available.
var Unavailable = errors.New("circuit-breaker: unavailable")

// CircuitBreaker represents the circuit breaker.
type CircuitBreaker struct {
	// State.
	status   int64
	counter  int64
	deadline time.Time
	total    int64

	// Options.
	success    int64
	failure    int64
	timeout    time.Duration
	now        func() time.Time
	errHandler func(error) bool
	errRate    float64
}

type Option struct {
	Success      int64
	Failure      int64
	Timeout      time.Duration
	Now          func() time.Time
	ErrorHandler func(err error) bool
	ErrorRate    float64
}

func NewOption() *Option {
	return &Option{
		Timeout: timeout,
		Success: success,
		Failure: failure,
		Now:     time.Now,
		ErrorHandler: func(err error) bool {
			return err != nil
		},
		ErrorRate: 0.90,
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
	}
}

// Exec updates the circuit breaker status based on the returned error.
func (cb *CircuitBreaker) Exec(ctx context.Context, h func(ctx context.Context) error) error {
	requestsTotal.Record(ctx, 1)

	if !cb.allow(ctx) {
		failuresTotal.Record(ctx, 1)
		return Unavailable
	}

	err := h(ctx)
	isErr := cb.errHandler(err)
	cb.update(!isErr)

	return err
}

// ResetIn returns the wait time before the service can be called again.
func (cb *CircuitBreaker) ResetIn() time.Duration {
	if cb.Status().IsOpen() {
		delta := cb.deadline.Sub(cb.now())
		if delta < 0 {
			return 0
		}

		return delta
	}

	return 0
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
		if cb.timerExpires() {
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
		if cb.incr() > cb.success {
			cb.reset()
			cb.transition(HalfOpen, Closed)
		}
	case Closed:
		total := cb.incrTotal()

		// If the failure threshold is not meet during the
		// time window, it expires.
		// We check if the total > 1, to prevent the first
		// call from being skipped.
		// This means this condition will only be reached when there is an error first.
		if total > 1 && cb.timerExpires() {
			cb.reset()
			return
		}

		cb.startTimer()

		// The service is healthy.
		if ok {
			return
		}

		// The service is unhealthy.
		// After a certain threshold, circuit breaker becomes
		// Open.
		failed := cb.incr()

		// Checking the error threshold is insufficient.
		// Additionally, we also check the min error rate.
		// An error rate of 90% means at least 9 out of 10
		// requests must be failing.
		// This prevents scenario where we have 100_000
		// success requests, but because the error threshold
		// is set to 5, the circuitbreaker trips.

		if isMinPercent(failed, total, cb.errRate) && failed > cb.failure {
			cb.transition(Closed, Open)
		}
	}
}

func (cb *CircuitBreaker) reset() {
	atomic.StoreInt64(&cb.total, 0)
	atomic.StoreInt64(&cb.counter, 0)
}

func (cb *CircuitBreaker) timerExpires() bool {
	return cb.now().After(cb.deadline)
}

func (cb *CircuitBreaker) startTimer() {
	cb.deadline = cb.now().Add(cb.timeout)
}

func (cb *CircuitBreaker) incrTotal() int64 {
	return atomic.AddInt64(&cb.total, 1)
}

func (cb *CircuitBreaker) incr() int64 {
	return atomic.AddInt64(&cb.counter, 1)
}

func (cb *CircuitBreaker) transition(from, to Status) {
	atomic.CompareAndSwapInt64(&cb.status, from.Int64(), to.Int64())
}

func isMinPercent(count, total int64, threshold float64) bool {
	return float64(count)/float64(total) >= threshold
}
