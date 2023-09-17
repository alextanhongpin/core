// Package circuitbreaker is an in-memory implementation of circuit breaker.
// The idea is that each local node (server) should maintain it's own knowledge
// of the service availability, instead of depending on external infrastructure
// like distributed cache.

package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

const (
	timeout   = 5 * time.Second
	success   = 5                // min 5 success before the circuit breaker becomes closed.
	failure   = 10               // min 10 failures before the circuit breaker becomes open.
	errRate   = 0.9              // at least 90% of the requests fails.
	errWindow = 10 * time.Second // time window to measure the error rate.
)

// errHandler checks if the error will cause the circuitbreaker to trip.
var errHandler = func(err error) bool {
	return err != nil
}

// Unavailable returns the error when the circuit breaker is not available.
var Unavailable = errors.New("circuit-breaker: unavailable")

type Option struct {
	// States.
	total        int64
	count        int64
	errTimer     time.Time
	timeoutTimer time.Time

	// Options.
	Success    int64
	Failure    int64
	Timeout    time.Duration
	Now        func() time.Time
	ErrHandler func(error) bool
	ErrRate    float64
	ErrWindow  time.Duration
}

func NewOption() *Option {
	return &Option{
		Success:    success,
		Failure:    failure,
		Timeout:    timeout,
		Now:        time.Now,
		ErrHandler: errHandler,
		ErrRate:    errRate,
		ErrWindow:  errWindow,
	}
}

// CircuitBreaker represents the circuit breaker.
type CircuitBreaker struct {
	mu     sync.RWMutex
	opt    *Option
	states [3]state
	state  Status
}

func New(opt *Option) *CircuitBreaker {
	if opt == nil {
		opt = NewOption()
	}

	return &CircuitBreaker{
		opt: opt,
		states: [3]state{
			NewClosedState(opt),
			NewOpenState(opt),
			NewHalfOpenState(opt),
		},
	}
}

func (cb *CircuitBreaker) ResetIn() time.Duration {
	cb.mu.RLock()
	status := cb.state
	cb.mu.RUnlock()
	if status.IsOpen() {
		delta := cb.opt.timeoutTimer.Sub(cb.opt.Now())
		if delta > 0 {
			return delta
		}

		return 0
	}

	return 0

}

func (cb *CircuitBreaker) Status() Status {
	cb.mu.RLock()
	status := cb.state
	cb.mu.RUnlock()

	return status
}

func (cb *CircuitBreaker) Do(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, ok := cb.states[cb.state].Next()
	if ok {
		cb.state = state
		cb.states[cb.state].Entry()
	}

	return cb.states[cb.state].Do(fn)
}

type state interface {
	Next() (Status, bool)
	Entry()
	Do(func() error) error
}

type ClosedState struct {
	opt *Option
}

func NewClosedState(opt *Option) *ClosedState {
	return &ClosedState{opt}
}

func (c *ClosedState) Entry() {
	c.resetFailureCounter()
}

func (c *ClosedState) Next() (Status, bool) {
	return Open, c.isFailureThresholdReached()
}

func (c *ClosedState) Do(fn func() error) error {
	// Success.
	err := fn()
	c.incrementFailureCounter(err)

	return err
}

func (c *ClosedState) resetFailureCounter() {
	c.opt.count = 0
}

func (c *ClosedState) isFailureThresholdReached() bool {
	o := c.opt

	// The state transition is only valid if the failures
	// count and error rate exceeds the threshold within the
	// error time window.
	if o.Now().After(o.errTimer) {
		return false
	}

	return o.count > o.Failure && rate(o.count, o.total) >= o.ErrRate
}

func (c *ClosedState) incrementFailureCounter(err error) {
	o := c.opt

	// If expired, reset the counter.
	if o.Now().After(o.errTimer) {
		o.total = 0
		o.count = 0
	}

	o.total++
	if o.ErrHandler(err) {
		o.count++
		if o.count == 1 {
			o.errTimer = o.Now().Add(o.ErrWindow)
		}
	}
}

type OpenState struct {
	opt *Option
}

func NewOpenState(opt *Option) *OpenState {
	return &OpenState{opt}
}

func (s *OpenState) Entry() {
	s.startTimeoutTimer()
}

func (s *OpenState) Next() (Status, bool) {
	return HalfOpen, s.isTimeoutTimerExpired()
}

func (s *OpenState) Do(fn func() error) error {
	return Unavailable
}

func (s *OpenState) startTimeoutTimer() {
	s.opt.timeoutTimer = s.opt.Now().Add(s.opt.Timeout)
}

func (s *OpenState) isTimeoutTimerExpired() bool {
	return s.opt.Now().After(s.opt.timeoutTimer)
}

type HalfOpenState struct {
	opt    *Option
	failed bool
}

func NewHalfOpenState(opt *Option) *HalfOpenState {
	return &HalfOpenState{opt: opt}
}

func (s *HalfOpenState) Entry() {
	s.resetSuccessCounter()
}

func (s *HalfOpenState) Next() (Status, bool) {
	if s.isOperationFailed() {
		return Open, true
	}

	return Closed, s.isSuccessCountThresholdExceeded()
}

func (s *HalfOpenState) Do(fn func() error) error {
	err := fn()
	failed := s.opt.ErrHandler(err)
	s.failed = failed

	if !failed {
		s.incrementSuccessCounter()
	}

	return err
}

func (s *HalfOpenState) resetSuccessCounter() {
	s.opt.count = 0
}

func (s *HalfOpenState) isOperationFailed() bool {
	return s.failed
}

func (s *HalfOpenState) isSuccessCountThresholdExceeded() bool {
	return s.opt.count > s.opt.Success
}

func (s *HalfOpenState) incrementSuccessCounter() {
	s.opt.count++
}

func rate(n, total int64) float64 {
	return float64(n) / float64(total)
}
