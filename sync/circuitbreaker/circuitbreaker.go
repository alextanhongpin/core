package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
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

func DefaultConfig() Config {
	return Config{
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

// WithDefaults returns an effective copy. Zero thresholds and durations select
// defaults. Nil hooks select default weighting; use a zero-returning hook to
// disable extra weighting. Hook functions remain shared dependencies.
func (c Config) WithDefaults() Config {
	d := DefaultConfig()
	if c.FailureThreshold == 0 {
		c.FailureThreshold = d.FailureThreshold
	}
	if c.FailurePeriod == 0 {
		c.FailurePeriod = d.FailurePeriod
	}
	if c.SuccessThreshold == 0 {
		c.SuccessThreshold = d.SuccessThreshold
	}
	if c.SuccessPeriod == 0 {
		c.SuccessPeriod = d.SuccessPeriod
	}
	if c.OpenTimeout == 0 {
		c.OpenTimeout = d.OpenTimeout
	}
	if c.FailureCount == nil {
		c.FailureCount = d.FailureCount
	}
	if c.SlowCallCount == nil {
		c.SlowCallCount = d.SlowCallCount
	}
	return c
}

// Validate checks effective configuration without mutating it.
func (c Config) Validate() error {
	if c.FailureThreshold <= 0 || c.SuccessThreshold <= 0 {
		return fmt.Errorf("circuitbreaker: thresholds must be positive")
	}
	if c.FailurePeriod <= 0 || c.SuccessPeriod <= 0 || c.OpenTimeout <= 0 {
		return fmt.Errorf("circuitbreaker: durations must be positive")
	}
	return nil
}

// CircuitBreaker is safe for concurrent calls. Its configuration is privately
// owned; callbacks remain shared and may execute concurrently. Construct with
// New; the zero value is not usable.
type CircuitBreaker struct {
	cfg           Config
	mu            sync.RWMutex
	counter       int
	counterExpiry time.Time
	status        Status
	generation    uint64
	timeout       time.Time
}

// New defaults and validates a configuration copy without starting work.
func New(cfg Config) (*CircuitBreaker, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &CircuitBreaker{cfg: cfg, status: Closed}, nil
}

// MustNew is New for startup wiring that must panic on invalid configuration.
func MustNew(cfg Config) *CircuitBreaker {
	cb, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return cb
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
			if cb.cfg.FailureCount != nil {
				failureCount += cb.cfg.FailureCount(err)
			}
			if cb.cfg.SlowCallCount != nil {
				failureCount += cb.cfg.SlowCallCount(time.Since(start))
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
		cb.timeout = time.Now().Add(cb.cfg.OpenTimeout)
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
	cb.timeout = time.Now().Add(cb.cfg.OpenTimeout)
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
		totalCount := cb.inc(successCount, cb.cfg.SuccessPeriod)

		// If success count threshold reached.
		if totalCount >= cb.cfg.SuccessThreshold {
			return cb.onClosed()
		}

		return HalfOpen
	}

	return cb.onOpened()
}

func (cb *CircuitBreaker) close(failureCount int) Status {
	// Increment failure counter.
	totalCount := cb.inc(failureCount, cb.cfg.FailurePeriod)

	// If failure threshold exceeded
	if totalCount >= cb.cfg.FailureThreshold {
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
