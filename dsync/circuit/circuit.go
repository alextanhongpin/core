// package circuit is an alternative of the classic circuitbreaker.
package circuit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	redis "github.com/redis/go-redis/v9"
)

type Status int

const (
	Closed Status = iota
	Disabled
	HalfOpen
	ForcedOpen
	Open
)

var statusText = map[Status]string{
	Closed:     "closed",
	Disabled:   "disabled",
	HalfOpen:   "half-open",
	ForcedOpen: "forced-open",
	Open:       "open",
}

func (s Status) String() string {
	return statusText[s]
}

func NewStatus(status string) Status {
	switch status {
	case Closed.String():
		return Closed
	case Disabled.String():
		return Disabled
	case HalfOpen.String():
		return HalfOpen
	case ForcedOpen.String():
		return ForcedOpen
	case Open.String():
		return Open
	default:
		return Closed
	}
}

const (
	breakDuration    = 5 * time.Second
	failureRatio     = 0.5              // at least 50% of the requests fails
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
	samplingDuration = 10 * time.Second // time window to measure the error rate.
	successThreshold = 5
)

var (
	ErrUnavailable = errors.New("circuit: unavailable")
	ErrForcedOpen  = errors.New("circuit: forced open")
)

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
	mu sync.RWMutex

	// State.
	status Status
	timer  *time.Timer

	// Option.
	FailureCount  func(error) int
	Now           func() time.Time
	SlowCallCount func(time.Duration) int
	channel       string
	client        *redis.Client
	counter       *rate.Errors
	opt           *Option
}

func New(client *redis.Client, channel string, opt *Option) (*Breaker, func()) {
	if opt == nil {
		opt = NewOption()
	}

	b := &Breaker{
		FailureCount: func(err error) int {
			// Ignore context cancellation.
			if errors.Is(err, context.Canceled) {
				return 0
			}

			// Deadlines are considered as failures.
			if errors.Is(err, context.DeadlineExceeded) {
				return 5
			}

			return 1
		},
		Now: time.Now,
		SlowCallCount: func(duration time.Duration) int {
			// Every 5th second, penalize the slow call.
			return int(duration / (5 * time.Second))
		},
		channel: channel,
		client:  client,
		counter: rate.NewErrors(opt.SamplingDuration),
		opt:     opt,
	}
	return b, b.init()
}

func (b *Breaker) init() func() {
	ctx := context.Background()
	status, _ := b.client.Get(ctx, b.channel).Result()
	b.transition(NewStatus(status))

	pubsub := b.client.Subscribe(ctx, b.channel)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for msg := range pubsub.Channel() {
			b.transition(NewStatus(msg.Payload))
		}
	}()

	return func() {
		pubsub.Close()
		wg.Wait()
	}
}

func (b *Breaker) Do(ctx context.Context, fn func() error) error {
	switch status := b.Status(); status {
	case Open:
		return b.opened()
	case HalfOpen:
		return b.halfOpened(ctx, fn)
	case Closed:
		return b.closed(ctx, fn)
	case Disabled:
		return fn()
	case ForcedOpen:
		return b.forcedOpen()
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
}

func (b *Breaker) Status() Status {
	b.mu.RLock()
	status := b.status
	b.mu.RUnlock()

	return status
}

func (b *Breaker) transition(status Status) {
	if b.Status() == status {
		return
	}

	switch status {
	case Open:
		b.open()
	case Closed:
		b.close()
	case HalfOpen:
		b.halfOpen()
	case Disabled:
		b.disable()
	case ForcedOpen:
		b.forceOpen()
	}
}

func (b *Breaker) canOpen(n int) bool {
	if n <= 0 {
		return false
	}

	return b.isUnhealthy(b.counter.Inc(-int64(n)))
}

func (b *Breaker) open() {
	duration, _ := b.client.PTTL(context.Background(), b.channel).Result()
	if duration <= 0 {
		duration = b.opt.BreakDuration
	}

	b.mu.Lock()
	b.status = Open
	b.counter.Reset()

	if b.timer != nil {
		b.timer.Stop()
	}

	b.timer = time.AfterFunc(duration, func() {
		b.halfOpen()
	})
	b.mu.Unlock()
}

func (b *Breaker) opened() error {
	return ErrUnavailable
}

func (b *Breaker) halfOpen() {
	b.mu.Lock()
	b.status = HalfOpen
	b.counter.Reset()
	b.timer = nil
	b.mu.Unlock()
}

func (b *Breaker) halfOpened(ctx context.Context, fn func() error) error {
	start := time.Now()
	if err := fn(); err != nil {
		b.open()

		return errors.Join(err, b.publish(ctx, Open))
	}

	n := b.SlowCallCount(b.Now().Sub(start))
	if b.canOpen(n) {
		b.open()

		return b.publish(ctx, Open)
	}

	if b.canClose() {
		b.close()
	}

	return nil
}

func (b *Breaker) canClose() bool {
	return b.isHealthy(b.counter.Inc(1))
}

func (b *Breaker) close() {
	b.mu.Lock()
	b.status = Closed
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) closed(ctx context.Context, fn func() error) error {
	start := time.Now()
	if err := fn(); err != nil {
		n := b.FailureCount(err)
		n += b.SlowCallCount(b.Now().Sub(start))
		if b.canOpen(n) {
			b.open()

			return errors.Join(err, b.publish(ctx, Open))
		}

		return err
	}

	n := b.SlowCallCount(b.Now().Sub(start))
	if b.canOpen(n) {
		b.open()

		return b.publish(ctx, Open)
	}

	b.counter.Inc(1)

	return nil
}

func (b *Breaker) forceOpen() {
	b.mu.Lock()
	b.status = ForcedOpen
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) forcedOpen() error {
	return ErrForcedOpen
}

func (b *Breaker) disable() {
	b.mu.Lock()
	b.status = Disabled
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) publish(ctx context.Context, status Status) error {
	setErr := b.client.Set(ctx, b.channel, status.String(), b.opt.BreakDuration).Err()
	pubErr := b.client.Publish(ctx, b.channel, status.String()).Err()
	return errors.Join(setErr, pubErr)
}

func (b *Breaker) isHealthy(successes, _ float64) bool {
	return math.Ceil(successes) >= float64(b.opt.SuccessThreshold)
}

func (b *Breaker) isUnhealthy(successes, failures float64) bool {
	isFailureRatioExceeded := failureRate(successes, failures) >= b.opt.FailureRatio
	isFailureThresholdExceeded := math.Ceil(failures) >= float64(b.opt.FailureThreshold)

	return isFailureRatioExceeded && isFailureThresholdExceeded
}

func failureRate(successes, failures float64) float64 {
	num := failures
	den := failures + successes
	if den == 0.0 {
		return 0.0
	}

	return num / den
}
