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
	return statusText[s]
}

func ParseStatus(status string) Status {
	for k, v := range statusText {
		if v == status {
			return k
		}
	}
	return Unknown
}

var statusText = map[Status]string{
	Unknown:    "unknown",
	Closed:     "closed",
	HalfOpen:   "half-open",
	Opened:     "opened",
	Disabled:   "disabled",
	ForcedOpen: "forced-open",
}

const (
	Unknown Status = iota
	Closed
	HalfOpen
	Opened
	Disabled
	ForcedOpen
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
	}
}

// CircuitBreaker ...
type CircuitBreaker struct {
	*Options
	mu      sync.RWMutex
	timeout time.Time
	status  Status
	counter *TTL
}

func New(opts *Options) *CircuitBreaker {
	o := cmp.Or(opts, NewOptions())
	return &CircuitBreaker{
		Options: o,
		status:  Closed,
		counter: &TTL{},
	}
}

type handler[K, V any] = func(context.Context, K) (V, error)

func (cb *CircuitBreaker) Handler[K, V any](fn handler[K, V]) handler[K, V] {
	return func(ctx context.Context, req K) (V, error) {
		return cb.Do(ctx, fn, req)
	}
}

func (cb *CircuitBreaker) Do[K, V any](ctx context.Context, fn func(context.Context, K) (V, error), req K) (V, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	var zero V

	status := cb.begin()
	switch status {
	case Closed, HalfOpen:
		start := time.Now()
		res, err := fn(ctx, req)
		if err != nil || status == HalfOpen {
			cb.commit(err, time.Since(start))
		}
		return res, err
	case Opened, ForcedOpen:
		return zero, ErrOpened
	case Disabled:
		return fn(ctx, req)
	default:
		panic("unknown status")
	}
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
	if cb.status == Opened && !time.Now().Before(cb.timeout) {
		return cb.onHalfOpened()
	}

	return cb.status
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

func (cb *CircuitBreaker) onOpened() Status {
	cb.status = Opened
	cb.timeout = time.Now().Add(cb.OpenTimeout)
	return Opened
}

func (cb *CircuitBreaker) onClosed() Status {
	cb.status = Closed
	cb.counter.Reset()
	return Closed
}

func (cb *CircuitBreaker) onHalfOpened() Status {
	cb.status = HalfOpen
	cb.counter.Reset()
	cb.timeout = time.Time{}
	return HalfOpen
}

func (cb *CircuitBreaker) halfOpen(failureCount, successCount int) Status {
	// If success.
	if failureCount == 0 {
		// Increment success counter.
		cb.counter.Add(successCount)
		cb.counter.SetExpiry(cb.SuccessPeriod)

		// If success count threshold reached.
		if cb.counter.Load() >= cb.SuccessThreshold {
			return cb.onClosed()
		}

		return HalfOpen
	}

	return cb.onOpened()
}

func (cb *CircuitBreaker) close(failureCount int) Status {
	// Increment failure counter.
	cb.counter.Add(failureCount)
	cb.counter.SetExpiry(cb.FailurePeriod)

	// If failure threshold exceeded
	if cb.counter.Load() >= cb.FailureThreshold {
		return cb.onOpened()
	}

	return Closed
}
