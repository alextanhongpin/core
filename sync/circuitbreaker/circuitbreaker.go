package circuitbreaker

import (
	"cmp"
	"context"
	"errors"
	"sync"
	"time"
)

type Status int

func (s Status) Int() int {
	return int(s)
}

func (s Status) String() string {
	switch s {
	case Unknown:
		return "unknown"
	case Closed:
		return "closed"
	case HalfOpen:
		return "half-open"
	case Opened:
		return "opened"
	case Disabled:
		return "disabled"
	case ForcedOpen:
		return "forced-open"
	default:
		return "-"
	}
}

const (
	Unknown    Status = 0
	Closed     Status = 1
	HalfOpen   Status = 2
	Opened     Status = 3
	Disabled   Status = 4
	ForcedOpen Status = 5
)

var ErrOpened = errors.New("circuitbreaker: opened")

type Options struct {
	FailureThreshold int
	FailurePeriod    time.Duration
	SuccessThreshold int
	SuccessPeriod    time.Duration
	OpenTimeout      time.Duration
	FailureCount     func(cause error) int
	SlowCallCount    func(duration time.Duration) int
	Now              func() time.Time
}

func NewOptions() *Options {
	return &Options{
		FailureThreshold: 100,
		FailurePeriod:    time.Second,
		SuccessThreshold: 20,
		SuccessPeriod:    time.Second,
		OpenTimeout:      time.Minute,
		FailureCount: func(cause error) int {
			if errors.Is(cause, context.DeadlineExceeded) {
				return 2
			}
			return 0
		},
		SlowCallCount: func(duration time.Duration) int {
			if duration >= time.Minute {
				return 4
			}
			if duration >= 30*time.Second {
				return 2
			}
			if duration > time.Second {
				return 1
			}

			return 0
		},
		Now: time.Now,
	}
}

// CircuitBreaker ...
type CircuitBreaker struct {
	*Options
	mu      sync.RWMutex
	timeout time.Time
	status  Status
	counter int
}

func New(opts *Options) *CircuitBreaker {
	return &CircuitBreaker{
		Options: cmp.Or(opts, NewOptions()),
		status:  Closed,
	}
}

func (cb *CircuitBreaker) Do(ctx context.Context, fn func(context.Context) error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	status := cb.begin()
	switch status {
	case Closed, HalfOpen:
	case Opened, ForcedOpen:
		return ErrOpened
	case Disabled:
		return fn(ctx)
	default:
		panic("unknown status")
	}

	start := cb.Now()
	err := fn(ctx)
	if err != nil || status == HalfOpen {
		cb.commit(err, time.Since(start))
	}
	return err
}

func (cb *CircuitBreaker) SetStatus(status Status) {
	cb.mu.Lock()
	cb.status = status
	cb.mu.Unlock()
}

func (cb *CircuitBreaker) Status() Status {
	cb.mu.RLock()
	status := cb.status
	cb.mu.RUnlock()
	return status
}

func (cb *CircuitBreaker) begin() Status {
	status := cb.status
	timeout := cb.timeout

	if status == Opened && !cb.Now().Before(timeout) {
		return cb.onHalfOpened()
	}

	return status
}

func (cb *CircuitBreaker) commit(cause error, duration time.Duration) Status {
	var failureCount int
	var successCount int
	if cause != nil {
		failureCount = 1 + cb.FailureCount(cause) + cb.SlowCallCount(duration)
	} else {
		successCount = 1
	}

	status := cb.status
	switch status {
	case Closed:
		return cb.close(failureCount)
	case HalfOpen:
		return cb.halfOpen(failureCount, successCount)
	default:
		return Unknown
	}
}

func (cb *CircuitBreaker) onOpened(timeout time.Time) Status {
	cb.status = Opened
	cb.timeout = timeout
	return Opened
}

func (cb *CircuitBreaker) onClosed() Status {
	cb.status = Closed
	cb.counter = 0
	return Closed
}

func (cb *CircuitBreaker) onHalfOpened() Status {
	cb.status = HalfOpen
	cb.counter = 0
	cb.timeout = time.Time{}
	return HalfOpen
}

func (cb *CircuitBreaker) halfOpen(failureCount, successCount int) Status {
	// If success.
	if failureCount == 0 {
		// Increment success counter.
		cb.counter += successCount

		// If success count threshold reached.
		if cb.counter >= cb.SuccessThreshold {
			return cb.onClosed()
		}

		return HalfOpen
	}

	return cb.onOpened(cb.Now().Add(cb.OpenTimeout))
}

func (cb *CircuitBreaker) close(failureCount int) Status {
	// Increment failure counter.
	cb.counter += failureCount

	// If failure threshold exceeded
	if cb.counter >= cb.FailureThreshold {
		return cb.onOpened(cb.Now().Add(cb.OpenTimeout))
	}

	return Closed
}
