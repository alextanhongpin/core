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
	Isolated
	Open
)

var statusText = map[Status]string{
	Closed:   "closed",
	Disabled: "disabled",
	HalfOpen: "half-open",
	Isolated: "isolated",
	Open:     "open",
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
	case Isolated.String():
		return Isolated
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
	ErrIsolated    = errors.New("circuit: isolated")
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
	Now     func() time.Time
	channel string
	client  *redis.Client
	counter *rate.Errors
	opt     *Option
}

func New(client *redis.Client, channel string, opt *Option) (*Breaker, func()) {
	b := &Breaker{
		Now:     time.Now,
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
	case Isolated:
		return b.isolated()
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
}

// Escalate increases the failure threshold by `n` when the circuitbreaker is
// closed.
func (b *Breaker) Escalate(ctx context.Context, n int64) error {
	if b.Status() != Closed {
		return nil
	}

	return b.escalate(ctx, n)
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
	case Isolated:
		b.isolate()
	}
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
	if err := fn(); err != nil {
		return errors.Join(err, b.publish(ctx, Open))
	}

	if b.isHealthy(b.counter.Inc(1)) {
		b.close()
	}

	return nil
}

func (b *Breaker) close() {
	b.mu.Lock()
	b.status = Closed
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) closed(ctx context.Context, fn func() error) error {
	if err := fn(); err != nil {
		return errors.Join(err, b.Escalate(ctx, 1))
	}

	b.counter.Inc(1)

	return nil
}

func (b *Breaker) isolate() {
	b.mu.Lock()
	b.status = Isolated
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) isolated() error {
	return ErrIsolated
}

func (b *Breaker) disable() {
	b.mu.Lock()
	b.status = Disabled
	b.counter.Reset()
	b.mu.Unlock()
}

func (b *Breaker) escalate(ctx context.Context, n int64) error {
	if b.isTripped(b.counter.Inc(-n)) {
		return b.publish(ctx, Open)
	}

	return nil
}

func (b *Breaker) publish(ctx context.Context, status Status) error {
	setErr := b.client.Set(ctx, b.channel, status.String(), b.opt.BreakDuration).Err()
	pubErr := b.client.Publish(ctx, b.channel, status.String()).Err()
	return errors.Join(setErr, pubErr)
}

func (b *Breaker) isTripped(successes, failures float64) bool {
	isFailureRatioExceeded := failureRate(successes, failures) >= b.opt.FailureRatio
	isFailureThresholdExceeded := math.Ceil(failures) >= float64(b.opt.FailureThreshold)

	return isFailureRatioExceeded && isFailureThresholdExceeded
}

func (b *Breaker) isHealthy(successes, _ float64) bool {
	return math.Ceil(successes) >= float64(b.opt.SuccessThreshold)
}

func failureRate(successes, failures float64) float64 {
	num := failures
	den := failures + successes
	if den == 0.0 {
		return 0.0
	}

	return num / den
}
