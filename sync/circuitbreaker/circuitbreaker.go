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

type Status int64

const (
	StatusClosed Status = iota
	StatusOpen
	StatusHalfOpen
)

func (s Status) Int64() int64 {
	return int64(s)
}

var statusText = map[Status]string{
	StatusClosed:   "closed",
	StatusOpen:     "open",
	StatusHalfOpen: "half-open",
}

func (s Status) String() string {
	text := statusText[s]
	return text
}

// Group represents the circuit breaker.
type Group struct {
	// Private.
	status   int64
	counter  int64
	deadline time.Time

	// Public.
	Success  int64
	Failure  int64
	Timeout  time.Duration
	Now      func() time.Time
	Sampling rate.Sometimes
}

// New returns a pointer to Group.
func New() *Group {
	return &Group{
		Timeout:  timeout,
		Success:  success,
		Failure:  failure,
		Now:      time.Now,
		Sampling: rate.Sometimes{Every: 1},
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
	return g.deadline.Sub(g.Now())
}

func (g *Group) Status() Status {
	return Status(atomic.LoadInt64(&g.status))
}

func (g *Group) IsOpen() bool {
	return g.Status() == StatusOpen
}

func (g *Group) IsClosed() bool {
	return g.Status() == StatusClosed
}

func (g *Group) IsHalfOpen() bool {
	return g.Status() == StatusHalfOpen
}

func (g *Group) allow() bool {
	if g.IsOpen() {
		g.do(true)
	}

	return !g.IsOpen()
}

func (g *Group) do(ok bool) {
	g.Sampling.Do(func() {
		g.update(ok)
	})
}

func (g *Group) update(ok bool) {
	switch g.Status() {
	case StatusOpen:
		if g.Now().After(g.deadline) {
			atomic.StoreInt64(&g.counter, 0)
			atomic.CompareAndSwapInt64(&g.status, StatusOpen.Int64(), StatusHalfOpen.Int64())
		}
	case StatusHalfOpen:
		// The service is still unhealthy
		// Reset the counter and revert to Open.
		if !ok {
			atomic.StoreInt64(&g.counter, 0)
			atomic.CompareAndSwapInt64(&g.status, StatusHalfOpen.Int64(), StatusOpen.Int64())

			return
		}

		// The service is healthy.
		// After a certain threshold, circuit breaker becomes Closed.
		atomic.AddInt64(&g.counter, 1)
		if g.counter > g.Success {
			atomic.StoreInt64(&g.counter, 0)
			atomic.CompareAndSwapInt64(&g.status, StatusHalfOpen.Int64(), StatusClosed.Int64())
		}
	case StatusClosed:
		// The service is healthy.
		if ok {
			return
		}

		// The service is unhealthy.
		// After a certain threshold, circuit breaker becomes Open.
		atomic.AddInt64(&g.counter, 1)
		if g.counter > g.Failure {
			g.deadline = g.Now().Add(g.Timeout)
			atomic.CompareAndSwapInt64(&g.status, StatusClosed.Int64(), StatusOpen.Int64())
		}
	}
}
