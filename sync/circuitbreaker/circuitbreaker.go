package circuitbreaker

import (
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	timeout = 5 * time.Second
	success = 5  // 5 success before the circuit breaker becomes closed.
	failure = 10 // 10 failures before the circuit breaker becomes open.
)

// Unavailable returns the error when the circuit breaker is not available.
var Unavailable = errors.New("circuit-breaker: unavailable")

// Group represents the circuit breaker.
type Group struct {
	// State.
	status   int64
	counter  int64
	deadline time.Time

	// Options.
	success  int64
	failure  int64
	timeout  time.Duration
	now      func() time.Time
	sampling *rate.Sometimes
}

// New returns a pointer to Group.
func New() *Group {
	return &Group{
		timeout:  timeout,
		success:  success,
		failure:  failure,
		now:      time.Now,
		sampling: nil,
	}
}

// Do updates the circuit breaker status based on the returned error.
func (g *Group) Do(fn func() error) error {
	if !g.allow() {
		return Unavailable
	}

	err := fn()
	g.do(err == nil)

	return err
}

// ResetIn returns the wait time before the service can be called again.
func (g *Group) ResetIn() time.Duration {
	if g.Status().IsOpen() {
		return g.deadline.Sub(g.now())
	}

	return 0
}

func (g *Group) Status() Status {
	return Status(atomic.LoadInt64(&g.status))
}

func (g *Group) SetSuccessThreshold(n int64) {
	g.success = n
}

func (g *Group) SetFailureThreshold(n int64) {
	g.failure = n
}

func (g *Group) SetTimeout(timeout time.Duration) {
	g.timeout = timeout
}

func (g *Group) SetNow(now func() time.Time) {
	g.now = now
}

func (g *Group) SetSampling(sample *rate.Sometimes) {
	g.sampling = sample
}

func (g *Group) allow() bool {
	if g.Status().IsOpen() {
		g.do(true)
	}

	return !g.Status().IsOpen()
}

func (g *Group) do(ok bool) {
	if g.sampling == nil {
		g.update(ok)
		return
	}

	g.sampling.Do(func() {
		g.update(ok)
	})
}

func (g *Group) update(ok bool) {
	switch g.Status() {
	case Open:
		if g.timerExpires() {
			g.reset()
			g.transition(Open, HalfOpen)
		}
	case HalfOpen:
		// The service is still unhealthy
		// Reset the counter and revert to Open.
		if !ok {
			g.reset()
			g.transition(HalfOpen, Open)

			return
		}

		// The service is healthy.
		// After a certain threshold, circuit breaker becomes Closed.
		if g.incr() > g.success {
			g.reset()
			g.transition(HalfOpen, Closed)
		}
	case Closed:
		// The service is healthy.
		if ok {
			return
		}

		// The service is unhealthy.
		// After a certain threshold, circuit breaker becomes Open.
		if g.incr() > g.failure {
			g.startTimer()
			g.transition(Closed, Open)
		}
	}
}

func (g *Group) reset() {
	atomic.StoreInt64(&g.counter, 0)
}

func (g *Group) timerExpires() bool {
	return g.now().After(g.deadline)
}

func (g *Group) startTimer() {
	g.deadline = g.now().Add(g.timeout)
}

func (g *Group) incr() int64 {
	return atomic.AddInt64(&g.counter, 1)
}

func (g *Group) transition(from, to Status) {
	atomic.CompareAndSwapInt64(&g.status, from.Int64(), to.Int64())
}
