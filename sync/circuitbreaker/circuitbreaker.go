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

	"github.com/alextanhongpin/core/internal"
	"golang.org/x/exp/event"
	"golang.org/x/time/rate"
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

	// Options.
	handler  internal.CommandHandler
	success  int64
	failure  int64
	timeout  time.Duration
	now      func() time.Time
	sampling *rate.Sometimes
}

type Option struct {
	Handler  internal.CommandHandler
	Success  int64
	Failure  int64
	Timeout  time.Duration
	Now      func() time.Time
	Sampling *rate.Sometimes
}

func NewOption() *Option {
	return &Option{
		Timeout:  timeout,
		Success:  success,
		Failure:  failure,
		Now:      time.Now,
		Sampling: nil,
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

	if opt.Handler == nil {
		panic("circuitbreaker: missing handler in New")
	}

	return &CircuitBreaker{
		timeout:  opt.Timeout,
		success:  opt.Success,
		failure:  opt.Failure,
		now:      opt.Now,
		sampling: opt.Sampling,
		handler:  opt.Handler,
	}
}

// Exec updates the circuit breaker status based on the returned error.
func (cb *CircuitBreaker) Exec(ctx context.Context) error {
	requestsTotal.Record(ctx, 1)

	if !cb.allow(ctx) {
		failuresTotal.Record(ctx, 1)
		return Unavailable
	}

	err := cb.handler.Exec(ctx)
	cb.do(err == nil)

	return err
}

func (cb *CircuitBreaker) ExecFunc(ctx context.Context, h internal.CommandHandler) error {
	requestsTotal.Record(ctx, 1)

	if !cb.allow(ctx) {
		failuresTotal.Record(ctx, 1)
		return Unavailable
	}

	err := h.Exec(ctx)
	cb.do(err == nil)

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
		cb.do(true)
	}

	return !cb.Status().IsOpen()
}

func (cb *CircuitBreaker) do(ok bool) {
	if cb.sampling == nil {
		cb.update(ok)
		return
	}

	cb.sampling.Do(func() {
		cb.update(ok)
	})
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
		// The service is healthy.
		if ok {
			return
		}

		// The service is unhealthy.
		// After a certain threshold, circuit breaker becomes Open.
		if cb.incr() > cb.failure {
			cb.startTimer()
			cb.transition(Closed, Open)
		}
	}
}

func (cb *CircuitBreaker) reset() {
	atomic.StoreInt64(&cb.counter, 0)
}

func (cb *CircuitBreaker) timerExpires() bool {
	return cb.now().After(cb.deadline)
}

func (cb *CircuitBreaker) startTimer() {
	cb.deadline = cb.now().Add(cb.timeout)
}

func (cb *CircuitBreaker) incr() int64 {
	return atomic.AddInt64(&cb.counter, 1)
}

func (cb *CircuitBreaker) transition(from, to Status) {
	atomic.CompareAndSwapInt64(&cb.status, from.Int64(), to.Int64())
}

type circuit interface {
	ExecFunc(ctx context.Context, h internal.CommandHandler) error
}

func Exec[T any](ctx context.Context, cb circuit, handler internal.QueryHandler[T]) (v T, err error) {
	err = cb.ExecFunc(ctx, internal.CommandHandlerFunc(func(ctx context.Context) error {
		v, err = handler.Exec(ctx)
		return err
	}))

	return
}
