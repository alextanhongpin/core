package poll

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrEndOfQueue is returned when there are no more items to process
	ErrEndOfQueue = errors.New("poll: end of queue")
	// ErrEmptyQueue is returned when the queue is temporarily empty
	ErrEmptyQueue = errors.New("poll: empty queue")
	// ErrShutdown is returned when the poller is shutting down
	ErrShutdown = errors.New("poll: shutting down")

	// Backward compatibility
	EOQ   = ErrEndOfQueue
	Empty = ErrEmptyQueue
)

// PollOptions configures the polling behavior
type PollOptions struct {
	// BatchSize is the number of items to process in each batch
	BatchSize int

	// FailureThreshold is the number of consecutive failures before stopping
	FailureThreshold int

	// BackOff is the backoff strategy when no work is available
	BackOff func(idle int) time.Duration

	// MaxConcurrency is the maximum number of concurrent workers
	MaxConcurrency int

	// Timeout is the timeout for individual operations
	Timeout time.Duration

	// EventBufferSize is the size of the event channel buffer
	EventBufferSize int

	// OnError is called when an error occurs (optional)
	OnError func(error)

	// OnBatchComplete is called when a batch completes (optional)
	OnBatchComplete func(BatchMetrics)
}

// BatchMetrics contains metrics for a completed batch
type BatchMetrics struct {
	SuccessCount int
	FailureCount int
	Duration     time.Duration
	StartTime    time.Time
	EndTime      time.Time
}

// PollMetrics contains overall polling metrics
type PollMetrics struct {
	TotalBatches    int64
	TotalSuccess    int64
	TotalFailures   int64
	TotalIdleCycles int64
	StartTime       time.Time
	Running         bool
}

type Poll struct {
	// Configuration
	BatchSize        int
	FailureThreshold int
	BackOff          func(idle int) time.Duration
	MaxConcurrency   int
	Timeout          time.Duration
	EventBufferSize  int
	OnError          func(error)
	OnBatchComplete  func(BatchMetrics)

	// Internal state
	mu      sync.RWMutex
	running int32 // atomic

	// PollMetricsCollector interface for metrics collection
	metricsCollector PollMetricsCollector
}

// New creates a new Poll instance with default configuration
func New() *Poll {
	return NewWithOptions(PollOptions{})
}

// NewWithOptions creates a new Poll instance with custom options
func NewWithOptions(opts PollOptions) *Poll {
	p := &Poll{
		BatchSize:        opts.BatchSize,
		FailureThreshold: opts.FailureThreshold,
		BackOff:          opts.BackOff,
		MaxConcurrency:   opts.MaxConcurrency,
		Timeout:          opts.Timeout,
		EventBufferSize:  opts.EventBufferSize,
		OnError:          opts.OnError,
		OnBatchComplete:  opts.OnBatchComplete,
	}

	// Apply defaults
	if p.BatchSize <= 0 {
		p.BatchSize = 1_000
	}
	if p.FailureThreshold <= 0 {
		p.FailureThreshold = 25
	}
	if p.BackOff == nil {
		p.BackOff = ExponentialBackOff
	}
	if p.MaxConcurrency <= 0 {
		p.MaxConcurrency = MaxConcurrency()
	}
	if p.Timeout <= 0 {
		p.Timeout = 30 * time.Second
	}
	if p.EventBufferSize <= 0 {
		p.EventBufferSize = 100
	}

	// Initialize default metrics collector (atomic)
	p.metricsCollector = &AtomicPollMetricsCollector{}

	return p
}

// GetMetrics returns the current polling metrics
func (p *Poll) GetMetrics() PollMetrics {
	return p.metricsCollector.GetMetrics()
}

// IsRunning returns true if the poller is currently running
func (p *Poll) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// Poll starts polling with the given function and returns an event channel and stop function
func (p *Poll) Poll(fn func(context.Context) error) (<-chan Event, func()) {
	return p.PollWithContext(context.Background(), fn)
}

// PollWithContext starts polling with context and returns an event channel and stop function
func (p *Poll) PollWithContext(ctx context.Context, fn func(context.Context) error) (<-chan Event, func()) {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		// Already running, return closed channel
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}

	var (
		batchSize        = p.BatchSize
		ch               = make(chan Event, p.EventBufferSize)
		done             = make(chan struct{})
		failureThreshold = p.FailureThreshold
		backoff          = p.BackOff
		maxConcurrency   = p.MaxConcurrency
		timeout          = p.Timeout
	)

	// Initialize metrics
	p.mu.Lock()
	p.metricsCollector.SetStartTime(time.Now())
	p.metricsCollector.SetRunning(true)
	p.mu.Unlock()

	batch := func(batchCtx context.Context) (err error) {
		// Create timeout context if timeout is configured
		if timeout > 0 {
			var cancel context.CancelFunc
			batchCtx, cancel = context.WithTimeout(batchCtx, timeout)
			defer cancel()
		}

		limiter := NewLimiter(failureThreshold)
		startTime := time.Now()

		work := func() error {
			// Check if we're shutting down
			select {
			case <-done:
				return ErrShutdown
			case <-batchCtx.Done():
				return batchCtx.Err()
			default:
			}

			err := limiter.Do(func() error {
				return fn(batchCtx)
			})

			if errors.Is(err, ErrEndOfQueue) || errors.Is(err, ErrLimitExceeded) {
				return err
			}

			if err != nil && !errors.Is(err, ErrShutdown) {
				// Send error event (non-blocking)
				select {
				case <-done:
				case ch <- Event{
					Name: "error",
					Err:  err,
					Time: time.Now(),
				}:
				default:
					// Channel full, call error handler if available
					if p.OnError != nil {
						p.OnError(err)
					}
				}
			}

			// Failure in one batch should not stop the entire process.
			return nil
		}

		defer func(start time.Time) {
			endTime := time.Now()
			duration := endTime.Sub(start)

			metrics := BatchMetrics{
				SuccessCount: limiter.SuccessCount(),
				FailureCount: limiter.FailureCount(),
				Duration:     duration,
				StartTime:    start,
				EndTime:      endTime,
			}

			// Update global metrics
			p.metricsCollector.AddTotalSuccess(int64(metrics.SuccessCount))
			p.metricsCollector.AddTotalFailures(int64(metrics.FailureCount))

			// Call batch complete handler if available
			if p.OnBatchComplete != nil {
				p.OnBatchComplete(metrics)
			}

			// Send batch event (non-blocking)
			batchEvent := Event{
				Name: "batch",
				Data: map[string]any{
					"success":  metrics.SuccessCount,
					"failures": metrics.FailureCount,
					"total":    metrics.SuccessCount + metrics.FailureCount,
					"start":    start,
					"took":     duration.Seconds(),
				},
				Err:  err,
				Time: endTime,
			}

			select {
			case <-done:
			case ch <- batchEvent:
			default:
				// Channel full, drop event
			}
		}(startTime)

		// Do one work before starting the batch.
		// This allows us to check if the queue is empty.
		if err := work(); err != nil {
			if errors.Is(err, ErrShutdown) {
				return err
			}
			return fmt.Errorf("%w: %w", ErrEmptyQueue, err)
		}

		g, workCtx := errgroup.WithContext(batchCtx)
		g.SetLimit(maxConcurrency)

	loop:
		// Minus one work done earlier.
		for range batchSize - 1 {
			select {
			case <-done:
				break loop
			case <-workCtx.Done():
				break loop
			default:
				g.Go(work)
			}
		}

		return g.Wait()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(ch)
		defer atomic.StoreInt32(&p.running, 0)
		defer func() {
			p.mu.Lock()
			p.metricsCollector.SetRunning(false)
			p.mu.Unlock()
		}()

		var idle int
		for {
			// Check for shutdown or context cancellation
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			default:
			}

			// When the process is idle, we can sleep for a longer duration.
			sleep := backoff(idle)

			// Send poll event (non-blocking)
			pollEvent := Event{
				Name: "poll",
				Data: map[string]any{
					"idle":  idle,
					"sleep": sleep.Seconds(),
				},
				Time: time.Now(),
			}

			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case ch <- pollEvent:
			default:
				// Channel full, continue anyway
			}

			// Sleep with cancellation support
			timer := time.NewTimer(sleep)
			select {
			case <-done:
				timer.Stop()
				return
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				// Continue to batch processing
			}

			// Update idle cycles metric
			p.metricsCollector.IncTotalIdleCycles()

			if err := batch(ctx); err != nil {
				// Queue is empty, increment idle.
				if errors.Is(err, ErrEmptyQueue) {
					idle++
					continue
				}

				// End of queue, reset the idle counter.
				if errors.Is(err, ErrEndOfQueue) {
					idle = 0
					continue
				}

				// Shutdown requested
				if errors.Is(err, ErrShutdown) {
					return
				}

				// Context cancelled
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}

				// Too many failures, stop the process.
				return
			}

			// No errors, reset the idle counter.
			idle = 0
		}
	}()

	stopFunc := sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})

	return ch, stopFunc
}

// SetMetricsCollector allows injecting a custom PollMetricsCollector (e.g., Prometheus).
func (p *Poll) SetMetricsCollector(collector PollMetricsCollector) {
	if collector != nil {
		p.metricsCollector = collector
	}
}

// ExponentialBackOff returns the duration to sleep before the next batch.
// Idle will be zero if there are items in the queue. Otherwise, it will
// increment.
func ExponentialBackOff(idle int) time.Duration {
	idle = min(idle, 6) // Up to 2^6 = 64 seconds
	seconds := math.Pow(2, float64(idle))
	return time.Duration(seconds) * time.Second
}

// LinearBackOff provides linear backoff with configurable step and max
func LinearBackOff(step, max time.Duration) func(int) time.Duration {
	return func(idle int) time.Duration {
		duration := time.Duration(idle) * step
		if duration > max {
			return max
		}
		if duration < step {
			return step
		}
		return duration
	}
}

// ConstantBackOff provides constant backoff duration
func ConstantBackOff(duration time.Duration) func(int) time.Duration {
	return func(idle int) time.Duration {
		return duration
	}
}

// CustomExponentialBackOff provides configurable exponential backoff
func CustomExponentialBackOff(base time.Duration, multiplier float64, max time.Duration) func(int) time.Duration {
	return func(idle int) time.Duration {
		if idle == 0 {
			return base
		}
		duration := time.Duration(float64(base) * math.Pow(multiplier, float64(idle)))
		if duration > max {
			return max
		}
		return duration
	}
}

// MaxConcurrency returns the optimal concurrency level for the current system
func MaxConcurrency() int {
	return min(runtime.GOMAXPROCS(0), runtime.NumCPU())
}

// Event represents a polling event with metadata
type Event struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data,omitempty"`
	Err  error          `json:"error,omitempty"`
	Time time.Time      `json:"time"`
}

// String returns a string representation of the event
func (e Event) String() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v (error: %v)", e.Time.Format(time.RFC3339), e.Name, e.Data, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Time.Format(time.RFC3339), e.Name, e.Data)
}

// IsError returns true if the event contains an error
func (e Event) IsError() bool {
	return e.Err != nil
}

// IsBatch returns true if the event is a batch completion event
func (e Event) IsBatch() bool {
	return e.Name == "batch"
}

// IsPoll returns true if the event is a poll event
func (e Event) IsPoll() bool {
	return e.Name == "poll"
}

// PollMetricsCollector defines the interface for collecting poll metrics.
type PollMetricsCollector interface {
	IncTotalBatches()
	AddTotalSuccess(n int64)
	AddTotalFailures(n int64)
	IncTotalIdleCycles()
	SetStartTime(t time.Time)
	SetRunning(running bool)
	GetMetrics() PollMetrics
}

// AtomicPollMetricsCollector is the default atomic-based metrics implementation.
type AtomicPollMetricsCollector struct {
	totalBatches    int64
	totalSuccess    int64
	totalFailures   int64
	totalIdleCycles int64
	startTime       atomic.Value // time.Time
	running         int32
}

func (m *AtomicPollMetricsCollector) IncTotalBatches()         { atomic.AddInt64(&m.totalBatches, 1) }
func (m *AtomicPollMetricsCollector) AddTotalSuccess(n int64)  { atomic.AddInt64(&m.totalSuccess, n) }
func (m *AtomicPollMetricsCollector) AddTotalFailures(n int64) { atomic.AddInt64(&m.totalFailures, n) }
func (m *AtomicPollMetricsCollector) IncTotalIdleCycles()      { atomic.AddInt64(&m.totalIdleCycles, 1) }
func (m *AtomicPollMetricsCollector) SetStartTime(t time.Time) { m.startTime.Store(t) }
func (m *AtomicPollMetricsCollector) SetRunning(running bool) {
	if running {
		atomic.StoreInt32(&m.running, 1)
	} else {
		atomic.StoreInt32(&m.running, 0)
	}
}
func (m *AtomicPollMetricsCollector) GetMetrics() PollMetrics {
	var startTime time.Time
	if v := m.startTime.Load(); v != nil {
		startTime = v.(time.Time)
	}
	return PollMetrics{
		TotalBatches:    atomic.LoadInt64(&m.totalBatches),
		TotalSuccess:    atomic.LoadInt64(&m.totalSuccess),
		TotalFailures:   atomic.LoadInt64(&m.totalFailures),
		TotalIdleCycles: atomic.LoadInt64(&m.totalIdleCycles),
		StartTime:       startTime,
		Running:         atomic.LoadInt32(&m.running) == 1,
	}
}

// PrometheusPollMetricsCollector implements PollMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusPollMetricsCollector struct {
	TotalBatches    prometheus.Counter
	TotalSuccess    prometheus.Counter
	TotalFailures   prometheus.Counter
	TotalIdleCycles prometheus.Counter
	StartTime       prometheus.Gauge
	Running         prometheus.Gauge
}

func (m *PrometheusPollMetricsCollector) IncTotalBatches()         { m.TotalBatches.Inc() }
func (m *PrometheusPollMetricsCollector) AddTotalSuccess(n int64)  { m.TotalSuccess.Add(float64(n)) }
func (m *PrometheusPollMetricsCollector) AddTotalFailures(n int64) { m.TotalFailures.Add(float64(n)) }
func (m *PrometheusPollMetricsCollector) IncTotalIdleCycles()      { m.TotalIdleCycles.Inc() }
func (m *PrometheusPollMetricsCollector) SetStartTime(t time.Time) {
	m.StartTime.Set(float64(t.Unix()))
}
func (m *PrometheusPollMetricsCollector) SetRunning(running bool) {
	if running {
		m.Running.Set(1)
	} else {
		m.Running.Set(0)
	}
}
func (m *PrometheusPollMetricsCollector) GetMetrics() PollMetrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return PollMetrics{}
}
