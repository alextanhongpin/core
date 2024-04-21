package circuitbreaker

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	breakDuration    = 5 * time.Second
	successThreshold = 5                // min 5 successThreshold before the circuit breaker becomes closed.
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
	failureRatio     = 0.5              // at least 50% of the requests fails.
	samplingDuration = 10 * time.Second // time window to measure the error rate.

	// Ops
	noop    = 0
	failure = 1
	success = 2
)

var (
	ErrBrokenCircuit   = errors.New("circuit-breaker: broken")
	ErrIsolatedCircuit = errors.New("circuit-breaker: isolated")
)

//go:embed circuitbreaker.lua
var lua string

var script = redis.NewScript(lua)

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

type Option struct {
	SuccessThreshold int
	FailureThreshold int
	BreakDuration    time.Duration
	FailureRatio     float64
	SamplingDuration time.Duration
}

func NewOption() *Option {
	return &Option{
		SuccessThreshold: successThreshold,
		FailureThreshold: failureThreshold,
		BreakDuration:    breakDuration,
		FailureRatio:     failureRatio,
		SamplingDuration: samplingDuration,
	}
}

type CircuitBreaker struct {
	client         *redis.Client
	opt            *Option
	OnStateChanged func(ctx context.Context, from, to Status)
	Now            func() time.Time
}

func New(client *redis.Client, opt *Option) *CircuitBreaker {
	if opt == nil {
		opt = NewOption()
	}

	return &CircuitBreaker{
		client:         client,
		opt:            opt,
		OnStateChanged: func(ctx context.Context, from, to Status) {},
		Now:            time.Now,
	}
}

func (cb *CircuitBreaker) Status(ctx context.Context, key string) (Status, error) {
	status, err := cb.client.HGet(ctx, key, "status").Int()
	if errors.Is(err, redis.Nil) {
		return Closed, nil
	}

	if err != nil {
		return 0, err
	}

	return Status(status), nil
}

func (cb *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	if err := cb.eval(ctx, key, noop); err != nil {
		return err
	}

	if err := fn(); err != nil {
		return errors.Join(err, cb.eval(ctx, key, failure))
	} else {
		return cb.eval(ctx, key, success)
	}
}

func (cb *CircuitBreaker) eval(ctx context.Context, key string, state int) error {
	keys := []string{key}
	argv := []any{
		fmt.Sprint(cb.Now().UnixNano()),
		fmt.Sprint(int(cb.opt.SamplingDuration)),
		fmt.Sprint(cb.opt.FailureThreshold),
		fmt.Sprint(cb.opt.FailureRatio),
		fmt.Sprint(cb.opt.SuccessThreshold),
		fmt.Sprint(int(cb.opt.BreakDuration)),
		fmt.Sprint(state),
	}
	statuses, err := script.Run(ctx, cb.client, keys, argv...).Int64Slice()
	if err != nil {
		return err
	}

	prev, next := Status(statuses[0]), Status(statuses[1])
	if prev != next {
		cb.OnStateChanged(ctx, prev, next)
	}

	switch next {
	case Open:
		return ErrBrokenCircuit
	case Isolated:
		return ErrIsolatedCircuit
	default:
	}

	return nil
}
