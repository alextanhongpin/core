package circuitbreaker

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	breakDuration    = 5 * time.Second
	failureRatio     = 0.5              // at least 50% of the requests fails.
	failureThreshold = 10               // min 10 failure before the circuit breaker becomes open.
	samplingDuration = 10 * time.Second // time window to measure the error rate.
	successThreshold = 5                // min 5 successThreshold before the circuit breaker becomes closed.
)

var ErrBrokenCircuit = errors.New("circuit-breaker: broken")

// Metrics contains runtime metrics for the circuit breaker.
type Metrics struct {
	TotalRequests      int64  // Total number of requests made
	SuccessfulRequests int64  // Number of successful requests
	FailedRequests     int64  // Number of failed requests
	RejectedRequests   int64  // Number of requests rejected due to open circuit
	StateTransitions   int64  // Number of state transitions
	CurrentState       string // Current state as string
}

// Options configures the circuit breaker behavior.
type Options struct {
	// BreakDuration is how long the circuit breaker stays open before transitioning to half-open.
	BreakDuration time.Duration

	// FailureRatio is the ratio of failures that triggers the circuit breaker to open.
	// Must be between 0.0 and 1.0. Default is 0.5 (50%).
	FailureRatio float64

	// FailureThreshold is the minimum number of failures before the circuit breaker can open.
	// Default is 10.
	FailureThreshold int

	// SamplingDuration is the time window to measure the error rate.
	// Default is 10 seconds.
	SamplingDuration time.Duration

	// SuccessThreshold is the minimum number of successes in half-open state before closing.
	// Default is 5.
	SuccessThreshold int

	// FailureCount is a function that returns the penalty count for a given error.
	// Default behavior ignores context cancellation and gives heavier penalty for timeouts.
	FailureCount func(error) int

	// SlowCallCount is a function that returns the penalty count for slow calls.
	// Default behavior gives 1 penalty point per 5 seconds of execution time.
	SlowCallCount func(time.Duration) int

	// OnStateChange is called when the circuit breaker changes state.
	OnStateChange func(old, new Status)

	// OnRequest is called before each request is processed.
	OnRequest func()

	// OnSuccess is called after each successful request.
	OnSuccess func(duration time.Duration)

	// OnFailure is called after each failed request.
	OnFailure func(err error, duration time.Duration)

	// OnReject is called when a request is rejected due to open circuit.
	OnReject func()
}

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

// CircuitBreakerMetricsCollector defines the interface for collecting circuit breaker metrics.
type CircuitBreakerMetricsCollector interface {
	IncTotalRequests()
	IncSuccessfulRequests()
	IncFailedRequests()
	IncRejectedRequests()
	IncStateTransitions()
	SetCurrentState(state string)
	GetMetrics() Metrics
}

// AtomicCircuitBreakerMetricsCollector is the default atomic-based metrics implementation.
type AtomicCircuitBreakerMetricsCollector struct {
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	rejectedRequests   int64
	stateTransitions   int64
	currentState       atomic.Value // string
}

func (m *AtomicCircuitBreakerMetricsCollector) IncTotalRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}
func (m *AtomicCircuitBreakerMetricsCollector) IncSuccessfulRequests() {
	atomic.AddInt64(&m.successfulRequests, 1)
}
func (m *AtomicCircuitBreakerMetricsCollector) IncFailedRequests() {
	atomic.AddInt64(&m.failedRequests, 1)
}
func (m *AtomicCircuitBreakerMetricsCollector) IncRejectedRequests() {
	atomic.AddInt64(&m.rejectedRequests, 1)
}
func (m *AtomicCircuitBreakerMetricsCollector) IncStateTransitions() {
	atomic.AddInt64(&m.stateTransitions, 1)
}
func (m *AtomicCircuitBreakerMetricsCollector) SetCurrentState(state string) {
	m.currentState.Store(state)
}
func (m *AtomicCircuitBreakerMetricsCollector) GetMetrics() Metrics {
	cs, _ := m.currentState.Load().(string)
	return Metrics{
		TotalRequests:      atomic.LoadInt64(&m.totalRequests),
		SuccessfulRequests: atomic.LoadInt64(&m.successfulRequests),
		FailedRequests:     atomic.LoadInt64(&m.failedRequests),
		RejectedRequests:   atomic.LoadInt64(&m.rejectedRequests),
		StateTransitions:   atomic.LoadInt64(&m.stateTransitions),
		CurrentState:       cs,
	}
}

// PrometheusCircuitBreakerMetricsCollector implements CircuitBreakerMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusCircuitBreakerMetricsCollector struct {
	TotalRequests      prometheus.Counter
	SuccessfulRequests prometheus.Counter
	FailedRequests     prometheus.Counter
	RejectedRequests   prometheus.Counter
	StateTransitions   prometheus.Counter
	CurrentState       prometheus.GaugeVec // label: state
}

func (m *PrometheusCircuitBreakerMetricsCollector) IncTotalRequests() { m.TotalRequests.Inc() }
func (m *PrometheusCircuitBreakerMetricsCollector) IncSuccessfulRequests() {
	m.SuccessfulRequests.Inc()
}
func (m *PrometheusCircuitBreakerMetricsCollector) IncFailedRequests()   { m.FailedRequests.Inc() }
func (m *PrometheusCircuitBreakerMetricsCollector) IncRejectedRequests() { m.RejectedRequests.Inc() }
func (m *PrometheusCircuitBreakerMetricsCollector) IncStateTransitions() { m.StateTransitions.Inc() }
func (m *PrometheusCircuitBreakerMetricsCollector) SetCurrentState(state string) {
	m.CurrentState.WithLabelValues(state).Set(1)
}
func (m *PrometheusCircuitBreakerMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}

// Breaker implements a circuit breaker with pluggable clock, hooks, and metrics.
type Breaker struct {
	// Configuration (copied from Options for performance).
	BreakDuration    time.Duration
	Counter          *rate.Errors
	FailureCount     func(error) int
	FailureRatio     float64
	FailureThreshold int
	SamplingDuration time.Duration
	SlowCallCount    func(time.Duration) int
	SuccessThreshold int

	// Callbacks
	OnStateChange func(old, new Status)
	OnRequest     func()
	OnSuccess     func(duration time.Duration)
	OnFailure     func(err error, duration time.Duration)
	OnReject      func()

	// Hooks and clock for testability.
	Now       func() time.Time
	AfterFunc func(time.Duration, func()) *time.Timer

	// State.
	mu            sync.RWMutex
	status        Status
	timer         *time.Timer
	probeInFlight bool

	// Metrics (using atomic operations for thread safety)
	metrics CircuitBreakerMetricsCollector
}

func New() *Breaker {
	return NewWithOptions(Options{})
}

// NewWithOptions creates a new circuit breaker with custom options.
func NewWithOptions(opts Options, metrics ...CircuitBreakerMetricsCollector) *Breaker {
	// Set defaults
	if opts.BreakDuration <= 0 {
		opts.BreakDuration = breakDuration
	}
	if opts.FailureRatio <= 0 {
		opts.FailureRatio = failureRatio
	}
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = failureThreshold
	}
	if opts.SamplingDuration <= 0 {
		opts.SamplingDuration = samplingDuration
	}
	if opts.SuccessThreshold <= 0 {
		opts.SuccessThreshold = successThreshold
	}
	if opts.FailureCount == nil {
		opts.FailureCount = defaultFailureCount
	}
	if opts.SlowCallCount == nil {
		opts.SlowCallCount = defaultSlowCallCount
	}

	var m CircuitBreakerMetricsCollector
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	} else {
		m = &AtomicCircuitBreakerMetricsCollector{}
	}
	b := &Breaker{
		BreakDuration:    opts.BreakDuration,
		Counter:          rate.NewErrors(opts.SamplingDuration),
		FailureCount:     opts.FailureCount,
		FailureRatio:     opts.FailureRatio,
		FailureThreshold: opts.FailureThreshold,
		SamplingDuration: opts.SamplingDuration,
		SlowCallCount:    opts.SlowCallCount,
		SuccessThreshold: opts.SuccessThreshold,
		OnStateChange:    opts.OnStateChange,
		OnRequest:        opts.OnRequest,
		OnSuccess:        opts.OnSuccess,
		OnFailure:        opts.OnFailure,
		OnReject:         opts.OnReject,
		Now:              time.Now,
		AfterFunc:        time.AfterFunc,
		status:           Closed,
		metrics:          m,
	}
	m.SetCurrentState(b.status.String())
	return b
}

func defaultFailureCount(err error) int {
	// Ignore context cancellation.
	if errors.Is(err, context.Canceled) {
		return 0
	}

	// Additional penalty for deadlines.
	if errors.Is(err, context.DeadlineExceeded) {
		return 5
	}

	return 1
}

func defaultSlowCallCount(duration time.Duration) int {
	// Every 5th second, penalty increases by 1.
	return int(duration / (5 * time.Second))
}

func (b *Breaker) Status() Status {
	b.mu.RLock()
	status := b.status
	b.mu.RUnlock()

	return status
}

// Metrics returns a copy of the current metrics.
func (b *Breaker) Metrics() Metrics {
	return b.metrics.GetMetrics()
}

func (b *Breaker) Do(fn func() error) error {
	b.metrics.IncTotalRequests()

	if b.OnRequest != nil {
		b.OnRequest()
	}

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

// setStatus transitions state, resets the counter and timer, and invokes a hook.
func (b *Breaker) setStatus(s Status) {
	b.mu.Lock()
	old := b.status
	b.status = s
	b.Counter.Reset()
	if b.timer != nil {
		b.timer.Stop()
	}
	hook := b.OnStateChange
	b.mu.Unlock()

	if old != s {
		b.metrics.IncStateTransitions()
		b.metrics.SetCurrentState(s.String())
		if hook != nil {
			go hook(old, s)
		}
	}
}

func (b *Breaker) canOpen(n int) bool {
	if n <= 0 {
		return false
	}

	_ = b.Counter.Failure().Add(float64(n))
	r := b.Counter.Rate()
	return b.isUnhealthy(r.Success(), r.Failure())
}

func (b *Breaker) open() {
	b.setStatus(Open)
	b.timer = b.AfterFunc(b.BreakDuration, func() {
		b.halfOpen()
	})
}

func (b *Breaker) opened() error {
	b.metrics.IncRejectedRequests()
	if b.OnReject != nil {
		b.OnReject()
	}
	return ErrBrokenCircuit
}

func (b *Breaker) canClose() bool {
	_ = b.Counter.Success().Inc()
	r := b.Counter.Rate()
	return b.isHealthy(r.Success(), r.Failure())
}

func (b *Breaker) close() {
	b.setStatus(Closed)
}

func (b *Breaker) closed(fn func() error) error {
	start := b.Now()
	err := fn()
	duration := b.Now().Sub(start)

	if err != nil {
		b.metrics.IncFailedRequests()
		if b.OnFailure != nil {
			b.OnFailure(err, duration)
		}

		n := b.FailureCount(err)
		n += b.SlowCallCount(duration)
		if b.canOpen(n) {
			b.open()
		}

		return err
	}

	b.metrics.IncSuccessfulRequests()
	if b.OnSuccess != nil {
		b.OnSuccess(duration)
	}

	n := b.SlowCallCount(duration)
	if b.canOpen(n) {
		b.open()
		return nil
	}

	b.Counter.Success().Inc()

	return nil
}

func (b *Breaker) halfOpen() {
	b.setStatus(HalfOpen)
}

func (b *Breaker) halfOpened(fn func() error) error {
	// Allow only one in-flight probe in half-open
	b.mu.Lock()
	if b.probeInFlight {
		b.mu.Unlock()
		b.metrics.IncRejectedRequests()
		if b.OnReject != nil {
			b.OnReject()
		}
		return ErrBrokenCircuit
	}
	b.probeInFlight = true
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.probeInFlight = false
		b.mu.Unlock()
	}()

	start := b.Now()
	err := fn()
	duration := b.Now().Sub(start)

	if err != nil {
		b.metrics.IncFailedRequests()
		if b.OnFailure != nil {
			b.OnFailure(err, duration)
		}
		b.open()
		return err
	}

	b.metrics.IncSuccessfulRequests()
	if b.OnSuccess != nil {
		b.OnSuccess(duration)
	}

	n := b.SlowCallCount(duration)
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
