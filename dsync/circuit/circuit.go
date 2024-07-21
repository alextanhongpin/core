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

func NewStatus(status string) Status {
	switch status {
	case Open.String():
		return Open
	case Closed.String():
		return Closed
	case HalfOpen.String():
		return HalfOpen
	case Isolated.String():
		return Isolated
	default:
		return Closed
	}
}

const (
	breakDuration    = 5 * time.Second
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
	failureRatio     = 0.5              // at least 50% of the requests fails
	samplingDuration = 10 * time.Second // time window to measure the error rate.
)

var ErrUnavailable = errors.New("circuit: unavailable")

type Option struct {
	BreakDuration    time.Duration
	FailureRatio     float64
	FailureThreshold int
	SamplingDuration time.Duration
}

func NewOption() *Option {
	return &Option{
		BreakDuration:    breakDuration,
		FailureRatio:     failureRatio,
		FailureThreshold: failureThreshold,
		SamplingDuration: samplingDuration,
	}
}

type Breaker struct {
	mu   sync.RWMutex
	once sync.Once

	// State.
	status Status
	t      *time.Timer

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
	var stop func() = func() {}

	b.once.Do(func() {
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

		stop = func() {
			pubsub.Close()
			wg.Wait()
		}
	})

	return stop
}

func (b *Breaker) Do(ctx context.Context, fn func() error) error {
	switch status := b.Status(); status {
	case Open:
		return b.opened()
	case HalfOpen:
		return b.halfOpened(ctx, fn)
	case Closed:
		return b.closed(ctx, fn)
	case Isolated:
		return fn()
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

	if b.t != nil {
		b.t.Stop()
	}

	b.t = time.AfterFunc(duration, func() {
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
	b.t = nil
	b.mu.Unlock()
}

func (b *Breaker) halfOpened(ctx context.Context, fn func() error) error {
	if err := fn(); err != nil {
		return errors.Join(err, b.publish(ctx, Open))
	}

	b.close()

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
		if b.isTripped(b.counter.Inc(-1)) {
			return errors.Join(err, b.publish(ctx, Open))
		}

		return err
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

func failureRate(successes, failures float64) float64 {
	num := failures
	den := failures + successes
	if den == 0.0 {
		return 0.0
	}

	return num / den
}
