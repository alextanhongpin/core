// package circuitbreaker is an alternative of the classic circuitbreaker.
package circuitbreaker

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
	ErrUnavailable = errors.New("circuit-breaker: unavailable")
	ErrForcedOpen  = errors.New("circuit-breaker: forced open")
)

type counter interface {
	Inc(successOrFailure int64) (successes, failures float64)
	Reset()
}

type CircuitBreaker struct {
	mu sync.RWMutex

	// State.
	status Status
	timer  *time.Timer

	// Options.
	BreakDuration     time.Duration
	FailureCount      func(error) int
	FailureRatio      float64
	FailureThreshold  int
	HeartbeatDuration time.Duration
	Now               func() time.Time
	SamplingDuration  time.Duration
	SlowCallCount     func(time.Duration) int
	SuccessThreshold  int

	// Dependencies.
	Counter counter
	channel string
	client  *redis.Client
}

func New(client *redis.Client, channel string) (*CircuitBreaker, func()) {
	b := &CircuitBreaker{
		BreakDuration:    breakDuration,
		FailureRatio:     failureRatio,
		FailureThreshold: failureThreshold,
		SamplingDuration: samplingDuration,
		SuccessThreshold: successThreshold,
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
		Counter: rate.NewErrors(samplingDuration),
	}
	return b, b.init()
}

func (b *CircuitBreaker) init() func() {
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

func (b *CircuitBreaker) Do(ctx context.Context, fn func() error) error {
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

func (b *CircuitBreaker) Status() Status {
	b.mu.RLock()
	status := b.status
	b.mu.RUnlock()

	return status
}

func (b *CircuitBreaker) transition(status Status) {
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

func (b *CircuitBreaker) canOpen(n int) bool {
	if n <= 0 {
		return false
	}

	return b.isUnhealthy(b.Counter.Inc(-int64(n)))
}

func (b *CircuitBreaker) open() {
	duration, _ := b.client.PTTL(context.Background(), b.channel).Result()
	if duration <= 0 {
		duration = b.BreakDuration
	}

	b.mu.Lock()
	b.status = Open
	b.Counter.Reset()

	if b.timer != nil {
		b.timer.Stop()
	}

	b.timer = time.AfterFunc(duration, func() {
		b.halfOpen()
	})
	b.mu.Unlock()
}

func (b *CircuitBreaker) opened() error {
	return ErrUnavailable
}

func (b *CircuitBreaker) halfOpen() {
	b.mu.Lock()
	b.status = HalfOpen
	b.Counter.Reset()
	b.timer = nil
	b.mu.Unlock()
}

func (b *CircuitBreaker) halfOpened(ctx context.Context, fn func() error) error {
	start := b.Now()
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

func (b *CircuitBreaker) canClose() bool {
	return b.isHealthy(b.Counter.Inc(1))
}

func (b *CircuitBreaker) close() {
	b.mu.Lock()
	b.status = Closed
	b.Counter.Reset()
	b.mu.Unlock()
}

func (b *CircuitBreaker) closed(ctx context.Context, fn func() error) error {
	start := time.Now()

	if d := b.HeartbeatDuration; d > 0 {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			t := time.NewTicker(d)

			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					if b.canOpen(b.SlowCallCount(d)) {
						b.open()

						return
					}
				}
			}
		}()
	}

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

	b.Counter.Inc(1)

	return nil
}

func (b *CircuitBreaker) forceOpen() {
	b.mu.Lock()
	b.status = ForcedOpen
	b.Counter.Reset()
	b.mu.Unlock()
}

func (b *CircuitBreaker) forcedOpen() error {
	return ErrForcedOpen
}

func (b *CircuitBreaker) disable() {
	b.mu.Lock()
	b.status = Disabled
	b.Counter.Reset()
	b.mu.Unlock()
}

func (b *CircuitBreaker) publish(ctx context.Context, status Status) error {
	setErr := b.client.Set(ctx, b.channel, status.String(), b.BreakDuration).Err()
	pubErr := b.client.Publish(ctx, b.channel, status.String()).Err()
	return errors.Join(setErr, pubErr)
}

func (b *CircuitBreaker) isHealthy(successes, _ float64) bool {
	return math.Ceil(successes) >= float64(b.SuccessThreshold)
}

func (b *CircuitBreaker) isUnhealthy(successes, failures float64) bool {
	isFailureRatioExceeded := failureRate(successes, failures) >= b.FailureRatio
	isFailureThresholdExceeded := math.Ceil(failures) >= float64(b.FailureThreshold)

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
