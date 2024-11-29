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

	// State.
	mu     sync.RWMutex
	status Status
	timer  *time.Timer
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

func (b *Breaker) canOpen(n int) bool {
	if n <= 0 {
		return false
	}

	res := b.Counter.AddFailure(float64(n))
	return b.isUnhealthy(res.Success, res.Failure)
}

func (b *Breaker) open() {
	b.mu.Lock()
	b.status = Open
	b.Counter.Reset()
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(b.BreakDuration, func() {
		b.halfOpen()
	})
	b.mu.Unlock()
}

func (b *Breaker) opened() error {
	return ErrBrokenCircuit
}

func (b *Breaker) canClose() bool {
	res := b.Counter.IncSuccess()
	return b.isHealthy(res.Success, res.Failure)
}

func (b *Breaker) close() {
	b.mu.Lock()
	b.status = Closed
	b.Counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) closed(fn func() error) error {
	start := time.Now()
	if err := fn(); err != nil {
		n := b.FailureCount(err)
		n += b.SlowCallCount(time.Since(start))
		if b.canOpen(n) {
			b.open()
		}

		return err
	}

	n := b.SlowCallCount(time.Since(start))
	if b.canOpen(n) {
		b.open()

		return nil
	}

	b.Counter.IncSuccess()

	return nil
}

func (b *Breaker) halfOpen() {
	b.mu.Lock()
	b.status = HalfOpen
	b.Counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) halfOpened(fn func() error) error {
	start := time.Now()
	if err := fn(); err != nil {
		b.open()

		return err
	}

	n := b.SlowCallCount(time.Since(start))
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
