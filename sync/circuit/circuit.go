package circuit

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

const (
	breakDuration    = 5 * time.Second
	failureRatio     = 0.5              // at least 50% of the requests fails.
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
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

type Option struct {
	BreakDuration    time.Duration
	FailureRatio     float64
	FailureThreshold int
	SamplingDuration time.Duration
	SuccessThreshold int
}

func NewOption() *Option {
	return &Option{
		BreakDuration:    breakDuration,
		FailureRatio:     failureRatio,
		FailureThreshold: failureThreshold,
		SamplingDuration: samplingDuration,
		SuccessThreshold: successThreshold,
	}
}

type Breaker struct {
	counter *rate.Errors
	mu      sync.RWMutex
	opt     *Option
	status  Status
	timer   *time.Timer
}

func New(opt *Option) *Breaker {
	if opt == nil {
		opt = NewOption()
	}

	return &Breaker{
		counter: rate.NewErrors(opt.SamplingDuration),
		opt:     opt,
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

func (b *Breaker) open() {
	b.mu.Lock()
	b.status = Open
	b.counter.Reset()
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(b.opt.BreakDuration, func() {
		b.halfOpen()
	})
	b.mu.Unlock()
}

func (b *Breaker) opened() error {
	return ErrBrokenCircuit
}

func (b *Breaker) close() {
	b.mu.Lock()
	b.status = Closed
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) closed(fn func() error) error {
	if err := fn(); err != nil {
		if b.isTripped(b.counter.Inc(-1)) {
			b.open()
		}

		return err
	}

	b.counter.Inc(1)

	return nil
}

func (b *Breaker) halfOpen() {
	b.mu.Lock()
	b.status = HalfOpen
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) halfOpened(fn func() error) error {
	if err := fn(); err != nil {
		b.open()
		return err
	}

	successes, _ := b.counter.Inc(1)
	if int(math.Ceil(successes)) >= b.opt.SuccessThreshold {
		b.close()
	}

	return nil
}

func (b *Breaker) isTripped(successes, failures float64) bool {
	isFailureRatioExceeded := failureRate(successes, failures) >= b.opt.FailureRatio
	isFailureThresholdExceeded := math.Ceil(failures) >= float64(b.opt.FailureThreshold)

	return isFailureRatioExceeded && isFailureThresholdExceeded
}

func failureRate(successes, failures float64) float64 {
	num := failures
	den := failures + successes
	if den <= 0 {
		return 0
	}

	return num / den
}
