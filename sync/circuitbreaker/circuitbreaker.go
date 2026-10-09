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

type Config struct {
	FailureThreshold int
	FailurePeriod    time.Duration
	SuccessThreshold int
	SuccessPeriod    time.Duration
	OpenTimeout      time.Duration
	FailureCount     func(cause error) int
	SlowCallCount    func(duration time.Duration) int
}

func DefaultConfig() *Config {
	return &Config{
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

var _ circuitbreaker = (*CircuitBreaker)(nil)

// CircuitBreaker is safe for concurrent calls. Configure the exported Config
// fields before use; callbacks may execute concurrently.
type CircuitBreaker struct {
	*Config
	mu            sync.RWMutex
	counter       int
	counterExpiry time.Time
	status        Status
	generation    uint64
	timeout       time.Time
}

func New(cfg *Config) *CircuitBreaker {
	return &CircuitBreaker{
		Config: cmp.Or(cfg, DefaultConfig()),
		status: Closed,
	}
}

func (cb *CircuitBreaker) Do(fn func() error) error {
	cb.mu.Lock()
	status := cb.begin()
	generation := cb.generation
	cb.mu.Unlock()

	switch status {
	case Closed, HalfOpen:
		start := time.Now()
		err := fn()
		failureCount, successCount := 0, 0
		if err != nil {
			failureCount = 1
			if cb.FailureCount != nil {
				failureCount += cb.FailureCount(err)
			}
			if cb.SlowCallCount != nil {
				failureCount += cb.SlowCallCount(time.Since(start))
			}
		} else {
			successCount = 1
		}
		cb.mu.Lock()
		// Results admitted by an older state must not change the current state.
		if cb.generation == generation {
			if status == HalfOpen {
				cb.halfOpen(failureCount, successCount)
			} else if err != nil {
				cb.close(failureCount)
			}
		}
		cb.mu.Unlock()
		return err
	case Opened, ForcedOpen:
		return ErrOpened
	case Disabled:
		return fn()
	default:
		panic("unknown status")
	}
}

func (cb *CircuitBreaker) SetStatus(status Status) {
	cb.mu.Lock()
	cb.generation++
	cb.counter = 0
	cb.counterExpiry = time.Time{}
	cb.status = status
	if status == Opened {
		cb.timeout = time.Now().Add(cb.OpenTimeout)
	}
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

func (cb *CircuitBreaker) onOpened() Status {
	cb.generation++
	cb.status = Opened
	cb.timeout = time.Now().Add(cb.OpenTimeout)
	return Opened
}

func (cb *CircuitBreaker) onClosed() Status {
	cb.generation++
	cb.status = Closed
	cb.counter = 0
	cb.counterExpiry = time.Time{}
	return Closed
}

func (cb *CircuitBreaker) onHalfOpened() Status {
	cb.generation++
	cb.status = HalfOpen
	cb.counter = 0
	cb.counterExpiry = time.Time{}
	cb.timeout = time.Time{}
	return HalfOpen
}

func (cb *CircuitBreaker) halfOpen(failureCount, successCount int) Status {
	// If success.
	if failureCount == 0 {
		// Increment success counter.
		totalCount := cb.inc(successCount, cb.SuccessPeriod)

		// If success count threshold reached.
		if totalCount >= cb.SuccessThreshold {
			return cb.onClosed()
		}

		return HalfOpen
	}

	return cb.onOpened()
}

func (cb *CircuitBreaker) close(failureCount int) Status {
	// Increment failure counter.
	totalCount := cb.inc(failureCount, cb.FailurePeriod)

	// If failure threshold exceeded
	if totalCount >= cb.FailureThreshold {
		return cb.onOpened()
	}

	return Closed
}

func (cb *CircuitBreaker) inc(count int, ttl time.Duration) int {
	if !time.Now().Before(cb.counterExpiry) {
		cb.counter = 0
	}
	cb.counter += count
	cb.counterExpiry = time.Now().Add(ttl)
	return cb.counter
}
