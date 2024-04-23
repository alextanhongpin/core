// package circuit is an alternative of the classic circuitbreaker.
package circuit

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/redis/go-redis/v9"
)

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
	mu sync.RWMutex
	// State.
	open    bool
	resetAt time.Time

	// Option.
	client  *redis.Client
	opt     *Option
	counter *rate.ErrorCounter
	Now     func() time.Time
}

func New(client *redis.Client, opt *Option) *Breaker {
	return &Breaker{
		client:  client,
		opt:     opt,
		counter: rate.NewErrorCounter(opt.SamplingDuration),
		Now:     time.Now,
	}
}

func (b *Breaker) Do(ctx context.Context, key string, fn func() error) error {
	if err := b.check(); err != nil {
		return err
	}

	if err := b.sync(ctx, key); err != nil {
		return err
	}

	if err := fn(); err != nil {
		b.counter.MarkFailure(1)
		b.eval(ctx, key)
		return err
	}

	b.counter.MarkSuccess(1)
	return nil
}

func (b *Breaker) check() error {
	b.mu.RLock()
	open, resetAt := b.open, b.resetAt
	b.mu.RUnlock()

	if !open {
		return nil
	}

	isExpired := !b.Now().Before(resetAt) // now >= resetAt
	if isExpired {
		return nil
	}

	return ErrUnavailable
}

func (b *Breaker) sync(ctx context.Context, key string) error {
	ttl, err := b.client.TTL(ctx, key).Result()
	if err != nil {
		return err
	}

	open := ttl > 0

	b.mu.Lock()
	if b.open != open {
		b.open = open
		if b.open {
			b.resetAt = b.Now().Add(ttl)
		}
	}
	b.mu.Unlock()

	if open {
		return ErrUnavailable
	}

	return nil
}

func (b *Breaker) eval(ctx context.Context, key string) error {
	isFailureRatioExceeded := b.counter.Rate() >= b.opt.FailureRatio
	isFailureThresholdExceeded := math.Round(b.counter.Failure()) >= float64(b.opt.FailureThreshold)

	if isFailureRatioExceeded && isFailureThresholdExceeded {
		return b.client.SetNX(ctx, key, b.Now(), b.opt.BreakDuration).Err()
	}

	return nil
}
