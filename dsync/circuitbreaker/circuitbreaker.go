// package circuitbreaker is an alternative of the classic circuitbreaker.
package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/prometheus/client_golang/prometheus"
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

// MetricsCollector defines the interface for collecting circuit breaker metrics.
type MetricsCollector interface {
	IncRequests()
	IncSuccesses()
	IncFailures()
	IncOpen()
	IncClose()
}

type AtomicCBMetrics struct {
	requests  int64
	successes int64
	failures  int64
	open      int64
	close     int64
}

func (m *AtomicCBMetrics) IncRequests()  { atomic.AddInt64(&m.requests, 1) }
func (m *AtomicCBMetrics) IncSuccesses() { atomic.AddInt64(&m.successes, 1) }
func (m *AtomicCBMetrics) IncFailures()  { atomic.AddInt64(&m.failures, 1) }
func (m *AtomicCBMetrics) IncOpen()      { atomic.AddInt64(&m.open, 1) }
func (m *AtomicCBMetrics) IncClose()     { atomic.AddInt64(&m.close, 1) }

// PrometheusCBMetrics implements MetricsCollector using prometheus metrics.
type PrometheusCBMetrics struct {
	Requests  prometheus.Counter
	Successes prometheus.Counter
	Failures  prometheus.Counter
	Open      prometheus.Counter
	Close     prometheus.Counter
}

func (m *PrometheusCBMetrics) IncRequests()  { m.Requests.Inc() }
func (m *PrometheusCBMetrics) IncSuccesses() { m.Successes.Inc() }
func (m *PrometheusCBMetrics) IncFailures()  { m.Failures.Inc() }
func (m *PrometheusCBMetrics) IncOpen()      { m.Open.Inc() }
func (m *PrometheusCBMetrics) IncClose()     { m.Close.Inc() }

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
	Counter *rate.Errors
	channel string
	client  *redis.Client

	// Metrics.
	metricsCollector MetricsCollector
}

// Config represents the circuit breaker configuration
type Config struct {
	BreakDuration     time.Duration
	FailureRatio      float64
	FailureThreshold  int
	SamplingDuration  time.Duration
	SuccessThreshold  int
	HeartbeatDuration time.Duration
	FailureCount      func(error) int
	SlowCallCount     func(time.Duration) int
	Now               func() time.Time
}

// New creates a new circuit breaker with default configuration.
func New(client *redis.Client, channel string, collectors ...MetricsCollector) (*CircuitBreaker, func()) {
	if client == nil {
		panic("circuitbreaker: client cannot be nil")
	}
	if channel == "" {
		panic("circuitbreaker: channel cannot be empty")
	}

	var collector MetricsCollector
	if len(collectors) > 0 && collectors[0] != nil {
		collector = collectors[0]
	} else {
		collector = &AtomicCBMetrics{}
	}

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
		channel:          channel,
		client:           client,
		Counter:          rate.NewErrors(samplingDuration),
		metricsCollector: collector,
	}

	// Validate configuration
	b.validate()

	return b, b.init()
}

// NewWithConfig creates a new circuit breaker with custom configuration
func NewWithConfig(client *redis.Client, channel string, config Config) (*CircuitBreaker, func()) {
	if client == nil {
		panic("circuitbreaker: client cannot be nil")
	}
	if channel == "" {
		panic("circuitbreaker: channel cannot be empty")
	}

	b := &CircuitBreaker{
		BreakDuration:     config.BreakDuration,
		FailureRatio:      config.FailureRatio,
		FailureThreshold:  config.FailureThreshold,
		SamplingDuration:  config.SamplingDuration,
		SuccessThreshold:  config.SuccessThreshold,
		HeartbeatDuration: config.HeartbeatDuration,
		FailureCount:      config.FailureCount,
		SlowCallCount:     config.SlowCallCount,
		Now:               config.Now,
		channel:           channel,
		client:            client,
		Counter:           rate.NewErrors(config.SamplingDuration),
	}

	// Set defaults for nil functions
	if b.FailureCount == nil {
		b.FailureCount = func(err error) int {
			// Ignore context cancellation.
			if errors.Is(err, context.Canceled) {
				return 0
			}

			// Deadlines are considered as failures.
			if errors.Is(err, context.DeadlineExceeded) {
				return 5
			}

			return 1
		}
	}

	if b.SlowCallCount == nil {
		b.SlowCallCount = func(duration time.Duration) int {
			// Every 5th second, penalize the slow call.
			return int(duration / (5 * time.Second))
		}
	}

	if b.Now == nil {
		b.Now = time.Now
	}

	// Validate configuration
	b.validate()

	return b, b.init()
}

func (b *CircuitBreaker) validate() {
	if b.BreakDuration <= 0 {
		panic("circuitbreaker: BreakDuration must be positive")
	}
	if b.FailureRatio < 0 || b.FailureRatio > 1 {
		panic("circuitbreaker: FailureRatio must be between 0 and 1")
	}
	if b.FailureThreshold <= 0 {
		panic("circuitbreaker: FailureThreshold must be positive")
	}
	if b.SamplingDuration <= 0 {
		panic("circuitbreaker: SamplingDuration must be positive")
	}
	if b.SuccessThreshold <= 0 {
		panic("circuitbreaker: SuccessThreshold must be positive")
	}
}

func (b *CircuitBreaker) init() func() {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize status from Redis
	if status, err := b.client.Get(ctx, b.channel).Result(); err == nil {
		b.transition(NewStatus(status))
	}

	pubsub := b.client.Subscribe(ctx, b.channel)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pubsub.Close()

		for {
			select {
			case msg, ok := <-pubsub.Channel():
				if !ok {
					return
				}
				b.transition(NewStatus(msg.Payload))
			case <-ctx.Done():
				return
			}
		}
	}()

	return func() {
		cancel()
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

// Disable sets the circuit breaker to disabled state, allowing all requests to pass through.
func (b *CircuitBreaker) Disable() {
	b.disable()
}

// ForceOpen sets the circuit breaker to forced open state, rejecting all requests.
func (b *CircuitBreaker) ForceOpen() {
	b.forceOpen()
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

	_ = b.Counter.Failure().Add(float64(n))
	r := b.Counter.Rate()
	return b.isUnhealthy(r.Success(), r.Failure())
}

func (b *CircuitBreaker) open() {
	ctx := context.Background()
	duration, err := b.client.PTTL(ctx, b.channel).Result()
	if err != nil || duration <= 0 {
		duration = b.BreakDuration
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.status = Open
	b.Counter.Reset()

	if b.timer != nil {
		b.timer.Stop()
	}

	b.timer = time.AfterFunc(duration, func() {
		b.halfOpen()
	})
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
	_ = b.Counter.Success().Inc()
	r := b.Counter.Rate()
	return b.isHealthy(r.Success(), r.Failure())
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
			defer t.Stop()

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

	b.Counter.Success().Inc()

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

	if setErr != nil && pubErr != nil {
		return errors.Join(setErr, pubErr)
	}
	if setErr != nil {
		return setErr
	}
	if pubErr != nil {
		return pubErr
	}
	return nil
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

// Example Prometheus integration
//
// import (
//   "github.com/prometheus/client_golang/prometheus"
//   "github.com/prometheus/client_golang/prometheus/promhttp"
//   "github.com/redis/go-redis/v9"
//   "github.com/alextanhongpin/core/dsync/circuitbreaker"
//   "net/http"
// )
//
// func main() {
//   requests := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_requests", Help: "Total circuit breaker requests."})
//   successes := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_successes", Help: "Circuit breaker successes."})
//   failures := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_failures", Help: "Circuit breaker failures."})
//   open := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_open", Help: "Circuit breaker opened."})
//   close := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_close", Help: "Circuit breaker closed."})
//   prometheus.MustRegister(requests, successes, failures, open, close)
//
//   metrics := &circuitbreaker.PrometheusCBMetrics{
//     Requests: requests,
//     Successes: successes,
//     Failures: failures,
//     Open: open,
//     Close: close,
//   }
//   rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//   cb, _ := circuitbreaker.New(rdb, "cb:myservice", metrics)
//   http.Handle("/metrics", promhttp.Handler())
//   http.ListenAndServe(":8080", nil)
// }
