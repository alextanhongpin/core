// package circuit is an alternative of the classic circuitbreaker.
package circuit

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	redis "github.com/redis/go-redis/v9"
)

const (
	payload = "open"

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
	open bool
	t    *time.Timer

	// Option.
	Now     func() time.Time
	channel string
	client  *redis.Client
	counter *rate.Errors
	opt     *Option
}

func New(client *redis.Client, channel string, opt *Option) *Breaker {
	return &Breaker{
		Now:     time.Now,
		channel: channel,
		client:  client,
		counter: rate.NewErrors(opt.SamplingDuration),
		opt:     opt,
	}
}

func (b *Breaker) Start() func() {
	var stop func() = func() {}

	b.once.Do(func() {
		pubsub := b.client.Subscribe(context.Background(), b.channel)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			for msg := range pubsub.Channel() {
				if msg.Payload != payload {
					continue
				}

				b.mu.Lock()
				b.open = true
				if b.t != nil {
					b.t.Stop()
				}
				b.t = time.AfterFunc(b.opt.BreakDuration, func() {
					b.mu.Lock()
					b.open = false
					b.t = nil
					b.mu.Unlock()
				})
				b.mu.Unlock()
			}
		}()

		stop = func() {
			pubsub.Close()
			wg.Wait()
		}
	})

	return stop
}

func (b *Breaker) Do(ctx context.Context, key string, fn func() error) error {
	b.mu.RLock()
	open := b.open
	b.mu.RUnlock()

	if open {
		return ErrUnavailable
	}

	if err := fn(); err != nil {
		if b.isTripped(b.counter.Inc(-1)) {
			pubErr := b.client.Publish(ctx, b.channel, payload).Err()
			return errors.Join(err, pubErr)
		}

		return err
	}

	b.counter.Inc(1)
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
	if den == 0.0 {
		return 0.0
	}

	return num / den
}
