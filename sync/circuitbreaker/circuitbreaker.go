package circuitbreaker

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

const (
	breakDuration    = 5 * time.Second
	failureRatio     = 0.5              // at least 50% of the requests fails.
	failureThreshold = 10               // min 10 failure before the circuit breaker becomes open.
	samplingDuration = 10 * time.Second // time window to measure the error rate.
	successThreshold = 5                // min 5 successThreshold before the circuit breaker becomes closed.
)

var ErrBrokenCircuit = errors.New("circuit-breaker: broken")

type Status int

const (
	Closed Status = iota
	HalfOpen
	Open
)

var statusText = map[Status]string{
	Closed:   "closed",
	HalfOpen: "half-open",
	Open:     "open",
}

func (s Status) String() string {
	return statusText[s]
}

// Breaker implements a circuit breaker with pluggable clock and hooks.
type Breaker struct {
	// Configuration.
	BreakDuration    time.Duration
	Counter          *rate.Errors
	FailureCount     func(error) int
	FailureRatio     float64
	FailureThreshold int
	SamplingDuration time.Duration
	SlowCallCount    func(time.Duration) int
	SuccessThreshold int

	// Hooks and clock for testability.
	Now           func() time.Time
	AfterFunc     func(time.Duration, func()) *time.Timer
	OnStateChange func(old, new Status)

	// State.
	mu            sync.RWMutex
	status        Status
	timer         *time.Timer
	probeInFlight bool
}

func New() *Breaker {
	return &Breaker{
		BreakDuration: breakDuration,
		Counter:       rate.NewErrors(samplingDuration),
		FailureCount: func(err error) int {
			// Ignore context cancellation.
			if errors.Is(err, context.Canceled) {
				return 0
			}

			// Additional penalty for deadlines.
			if errors.Is(err, context.DeadlineExceeded) {
				return 5
			}

			return 1
		},
		FailureRatio:     failureRatio,
		FailureThreshold: failureThreshold,
		SamplingDuration: samplingDuration,
		SlowCallCount: func(duration time.Duration) int {
			// Every 5th second, penalty increases by 1.
			return int(duration / (5 * time.Second))
		},
		SuccessThreshold: successThreshold,
		// Inject defaults for testability.
		Now:           time.Now,
		AfterFunc:     time.AfterFunc,
		OnStateChange: nil,
		status:        Closed,
	}
}

func (b *Breaker) Status() Status {
	b.mu.RLock()
	status := b.status
	b.mu.RUnlock()

	return status
}

func (b *Breaker) Do(fn func() error) error {
	switch b.Status() {
	case Open:
		return b.opened()
	case HalfOpen:
		return b.halfOpened(fn)
	case Closed:
		return b.closed(fn)
	default:
		panic("unknown state")
	}
}

// setStatus transitions state, resets the counter and timer, and invokes a hook.
func (b *Breaker) setStatus(s Status) {
	b.mu.Lock()
	old := b.status
	b.status = s
	b.Counter.Reset()
	if b.timer != nil {
		b.timer.Stop()
	}
	hook := b.OnStateChange
	b.mu.Unlock()

	if old != s && hook != nil {
		go hook(old, s)
	}
}

func (b *Breaker) canOpen(n int) bool {
	if n <= 0 {
		return false
	}

	_ = b.Counter.Failure().Add(float64(n))
	r := b.Counter.Rate()
	return b.isUnhealthy(r.Success(), r.Failure())
}

func (b *Breaker) open() {
	b.setStatus(Open)
	b.timer = b.AfterFunc(b.BreakDuration, func() {
		b.halfOpen()
	})
}

func (b *Breaker) opened() error {
	return ErrBrokenCircuit
}

func (b *Breaker) canClose() bool {
	_ = b.Counter.Success().Inc()
	r := b.Counter.Rate()
	return b.isHealthy(r.Success(), r.Failure())
}

func (b *Breaker) close() {
	b.setStatus(Closed)
}

func (b *Breaker) closed(fn func() error) error {
	start := b.Now()
	if err := fn(); err != nil {
		n := b.FailureCount(err)
		n += b.SlowCallCount(b.Now().Sub(start))
		if b.canOpen(n) {
			b.open()
		}

		return err
	}

	n := b.SlowCallCount(b.Now().Sub(start))
	if b.canOpen(n) {
		b.open()

		return nil
	}

	b.Counter.Success().Inc()

	return nil
}

func (b *Breaker) halfOpen() {
	b.setStatus(HalfOpen)
}

func (b *Breaker) halfOpened(fn func() error) error {
	// Allow only one in-flight probe in half-open
	b.mu.Lock()
	if b.probeInFlight {
		b.mu.Unlock()
		return ErrBrokenCircuit
	}
	b.probeInFlight = true
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.probeInFlight = false
		b.mu.Unlock()
	}()

	start := b.Now()
	if err := fn(); err != nil {
		b.open()
		return err
	}

	n := b.SlowCallCount(b.Now().Sub(start))
	if b.canOpen(n) {
		b.open()
		return nil
	}

	if b.canClose() {
		b.close()
	}

	return nil
}

func (b *Breaker) isHealthy(success, _ float64) bool {
	return math.Ceil(success) >= float64(b.SuccessThreshold)
}

func (b *Breaker) isUnhealthy(success, failure float64) bool {
	isFailureRatioExceeded := failureRate(success, failure) >= b.FailureRatio
	isFailureThresholdExceeded := math.Ceil(failure) >= float64(b.FailureThreshold)

	return isFailureRatioExceeded && isFailureThresholdExceeded
}

func failureRate(success, failure float64) float64 {
	num := failure
	den := failure + success
	if den <= 0 {
		return 0
	}

	return num / den
}
