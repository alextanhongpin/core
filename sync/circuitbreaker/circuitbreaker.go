// Package circuitbreaker is an in-memory implementation of circuit breaker.
// The idea is that each local node (server) should maintain it's own knowledge
// of the service availability, instead of depending on external infrastructure
// like distributed cache.
package circuitbreaker

import (
	"context"
	"errors"
	"time"

	"golang.org/x/exp/constraints"
)

const (
	breakDuration    = 5 * time.Second
	successThreshold = 5                // min 5 successThreshold before the circuit breaker becomes closed.
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
	failureRatio     = 0.5              // at least 50% of the requests fails.
	samplingDuration = 10 * time.Second // time window to measure the error rate.
)

var (
	ErrBrokenCircuit   = errors.New("circuit-breaker: broken")
	ErrIsolatedCircuit = errors.New("circuit-breaker: isolated")
)

type Status int

const (
	Closed Status = iota
	Open
	HalfOpen
	Isolated
)

var statusText = map[Status]string{
	Closed:   "closed",
	Open:     "open",
	HalfOpen: "half-open",
	Isolated: "isolated",
}

func (s Status) String() string {
	return statusText[s]
}

type store interface {
	Get(ctx context.Context, key string) (*State, error)
	Set(ctx context.Context, key string, res *State) error
}

type Option struct {
	SuccessThreshold int
	FailureThreshold int
	BreakDuration    time.Duration
	Now              func() time.Time
	FailureRatio     float64
	SamplingDuration time.Duration
	Store            store
}

func NewOption() *Option {
	return &Option{
		SuccessThreshold: successThreshold,
		FailureThreshold: failureThreshold,
		BreakDuration:    breakDuration,
		Now:              time.Now,
		FailureRatio:     failureRatio,
		SamplingDuration: samplingDuration,
		Store:            NewInMemory(),
	}
}

type CircuitBreaker struct {
	opt            *Option
	states         []state
	OnStateChanged func(ctx context.Context, from, to Status)
}

func New(opt *Option) *CircuitBreaker {
	if opt == nil {
		opt = NewOption()
	}

	return &CircuitBreaker{
		opt: opt,
		states: []state{
			&ClosedState{
				SamplingDuration: opt.SamplingDuration,
				FailureThreshold: opt.FailureThreshold,
				FailureRatio:     opt.FailureRatio,
				Now:              opt.Now,
			},
			&OpenState{
				BreakDuration: opt.BreakDuration,
				Now:           opt.Now,
			},
			&HalfOpenState{
				SuccessThreshold: opt.SuccessThreshold,
			},
			&IsolatedState{},
		},
		OnStateChanged: func(ctx context.Context, from, to Status) {},
	}
}

type State struct {
	status  Status // Old status
	Status  Status // New status
	Count   int    // success or failure count.
	Total   int
	CloseAt time.Time
	ResetAt time.Time
}

func (c *CircuitBreaker) Status(ctx context.Context, key string) (Status, error) {
	res, err := c.opt.Store.Get(ctx, key)
	if err != nil {
		return Closed, err
	}

	return res.Status, nil
}

func (c *CircuitBreaker) ResetIn(ctx context.Context, key string) time.Duration {
	res, err := c.opt.Store.Get(ctx, key)
	if err != nil {
		return 0
	}

	// If the circuit breaker is not open, return 0.
	if res.Status != Open {
		return 0
	}

	delta := res.CloseAt.Sub(c.opt.Now())
	if delta < 0 {
		return 0
	}

	return delta
}

func (c *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	s, err := c.opt.Store.Get(ctx, key)
	if err != nil {
		return err
	}

	prev := s.Status
	next, ok := c.states[prev].Next(s)
	if ok {
		c.OnStateChanged(ctx, prev, next)
		s = c.states[next].Entry()
		s.status = prev
	}

	return errors.Join(
		c.states[s.Status].Do(s, fn),
		c.opt.Store.Set(ctx, key, s),
	)
}

type state interface {
	Entry() *State
	Next(s *State) (Status, bool)
	Do(s *State, fn func() error) error
}

var _ state = (*ClosedState)(nil)

// Each state holds an option.
type ClosedState struct {
	SamplingDuration time.Duration
	FailureThreshold int
	FailureRatio     float64
	Now              func() time.Time
}

func (c *ClosedState) Entry() *State {
	return &State{
		Status:  Closed,
		ResetAt: c.Now().Add(c.SamplingDuration),
	}
}

func (c *ClosedState) Next(s *State) (Status, bool) {
	c.resetInterval(s)

	isFailureThresholdReached := s.Count >= c.FailureThreshold
	isFailureRateExceeded := Ratio(s.Count, s.Total) >= c.FailureRatio

	return Open, isFailureThresholdReached && isFailureRateExceeded
}

func (c *ClosedState) Do(s *State, fn func() error) error {
	err := fn()

	// Increment failure counter.
	c.resetInterval(s)
	s.Total++
	if err != nil {
		s.Count++
	}

	return err
}

func (c *ClosedState) resetInterval(s *State) {
	// The state transition is only valid if the failures
	// count and error rate exceeds the threshold within the
	// error time window.
	//
	// Now >= resetAt
	now := c.Now()
	isResetAtElapsed := !now.Before(s.ResetAt)
	if isResetAtElapsed {
		s.ResetAt = now.Add(c.SamplingDuration)
		// resetFailureCounter
		s.Count = 0
		s.Total = 0
	}
}

type OpenState struct {
	BreakDuration time.Duration
	Now           func() time.Time
}

var _ state = (*OpenState)(nil)

func (o *OpenState) Entry() *State {
	return &State{
		Status:  Open,
		CloseAt: o.Now().Add(o.BreakDuration),
	}
}

func (o *OpenState) Next(s *State) (Status, bool) {
	isTimeoutTimerExpired := !o.Now().Before(s.CloseAt)
	return HalfOpen, isTimeoutTimerExpired
}

func (o *OpenState) Do(_ *State, fn func() error) error {
	return ErrBrokenCircuit
}

type HalfOpenState struct {
	SuccessThreshold int
}

var _ state = (*HalfOpenState)(nil)

func (h *HalfOpenState) Entry() *State {
	return &State{
		Status: HalfOpen,
	}
}

func (h *HalfOpenState) Next(s *State) (Status, bool) {
	isOperationFailed := s.Count < 0
	if isOperationFailed {
		return Open, true
	}

	isSuccessCountThresholdExceeded := s.Count >= h.SuccessThreshold

	return Closed, isSuccessCountThresholdExceeded
}

func (h *HalfOpenState) Do(s *State, fn func() error) error {
	err := fn()

	if err == nil {
		// Increment success counter.
		s.Count++
	}

	return err
}

type IsolatedState struct {
}

var _ state = (*IsolatedState)(nil)

func NewIsolatedState() *IsolatedState {
	return &IsolatedState{}
}

func (s *IsolatedState) Entry() *State {
	return &State{
		Status: Isolated,
	}
}

func (i *IsolatedState) Next(s *State) (Status, bool) {
	return Isolated, false
}

func (i *IsolatedState) Do(s *State, fn func() error) error {
	return ErrIsolatedCircuit
}

func Ratio[T constraints.Integer](n, total T) float64 {
	return float64(n) / float64(total)
}
