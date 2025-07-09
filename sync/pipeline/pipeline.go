package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"golang.org/x/sync/semaphore"
	// Prometheus is only required if using PrometheusPipelineMetricsCollector
	"github.com/prometheus/client_golang/prometheus"
)

// Common errors
var (
	ErrInvalidBufferSize = errors.New("pipeline: buffer size must be positive")
	ErrInvalidWorkers    = errors.New("pipeline: number of workers must be positive")
	ErrInvalidTimeout    = errors.New("pipeline: timeout must be positive")
	ErrPipelineClosed    = errors.New("pipeline: pipeline is closed")
	ErrContextCanceled   = errors.New("pipeline: context canceled")
)

// PipelineOptions configures pipeline behavior
type PipelineOptions struct {
	// BufferSize is the default buffer size for intermediate channels
	BufferSize int

	// WorkerCount is the default number of workers for parallel operations
	WorkerCount int

	// Timeout is the default timeout for operations
	Timeout time.Duration

	// EnableMetrics enables collection of pipeline metrics
	EnableMetrics bool

	// OnError is called when errors occur in the pipeline
	OnError func(error)

	// OnPanic is called when panics occur in pipeline goroutines
	OnPanic func(interface{})
}

// DefaultOptions returns default pipeline options
func DefaultOptions() PipelineOptions {
	return PipelineOptions{
		BufferSize:    100,
		WorkerCount:   4,
		Timeout:       30 * time.Second,
		EnableMetrics: false,
	}
}

// Metrics contains pipeline execution metrics
type Metrics struct {
	ProcessedCount int64         // Total items processed
	ErrorCount     int64         // Total errors encountered
	PanicCount     int64         // Total panics recovered
	StartTime      time.Time     // Pipeline start time
	Duration       time.Duration // Total execution time
	ThroughputRate float64       // Items per second
	ErrorRate      float64       // Error rate (0-1)
}

// String returns a string representation of metrics
func (m Metrics) String() string {
	return fmt.Sprintf(
		"processed: %d, errors: %d, panics: %d, duration: %v, throughput: %.2f/s, error_rate: %.2f%%",
		m.ProcessedCount, m.ErrorCount, m.PanicCount, m.Duration,
		m.ThroughputRate, m.ErrorRate*100,
	)
}

// Result represents a pipeline operation result
type Result[T any] struct {
	Data T
	Err  error
}

// Unwrap returns the data and error from the result
func (r Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}

// IsError returns true if the result contains an error
func (r Result[T]) IsError() bool {
	return r.Err != nil
}

// IsSuccess returns true if the result is successful
func (r Result[T]) IsSuccess() bool {
	return r.Err == nil
}

// MakeResult creates a new result with data and error
func MakeResult[T any](data T, err error) Result[T] {
	return Result[T]{Data: data, Err: err}
}

// MakeSuccessResult creates a successful result
func MakeSuccessResult[T any](data T) Result[T] {
	return Result[T]{Data: data, Err: nil}
}

// MakeErrorResult creates an error result
func MakeErrorResult[T any](err error) Result[T] {
	var zero T
	return Result[T]{Data: zero, Err: err}
}

// ThroughputInfo contains throughput metrics
type ThroughputInfo struct {
	Total         int           // Total items processed
	TotalFailures int           // Total failures
	ErrorRate     float64       // Error rate (0-1)
	Rate          float64       // Items per second
	Duration      time.Duration // Time elapsed
}

// String returns a string representation of throughput info
func (t ThroughputInfo) String() string {
	return fmt.Sprintf(
		"total: %d, errors: %d (%.1f%%), rate: %.2f/s, duration: %v",
		t.Total, t.TotalFailures, t.ErrorRate*100, t.Rate, t.Duration,
	)
}

// RateInfo contains rate metrics
type RateInfo struct {
	Total int     // Total items processed
	Rate  float64 // Items per second
}

// String returns a string representation of rate info
func (r RateInfo) String() string {
	return fmt.Sprintf("total: %d, rate: %.2f/s", r.Total, r.Rate)
}

// SafeClose safely closes a channel, handling potential panics
func SafeClose[T any](ch chan T) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = false
		}
	}()
	close(ch)
	return true
}

// withRecovery wraps a function with panic recovery
func withRecovery(fn func(), onPanic func(interface{})) {
	defer func() {
		if r := recover(); r != nil {
			if onPanic != nil {
				onPanic(r)
			}
		}
	}()
	fn()
}

// Buffer creates a buffered channel stage
func Buffer[T any](size int, in <-chan T) <-chan T {
	if size <= 0 {
		panic(ErrInvalidBufferSize)
	}

	out := make(chan T, size)
	go func() {
		defer close(out)
		for v := range in {
			out <- v
		}
	}()
	return out
}

// Queue acts as an intermediary stage that queues results to a buffered channel
func Queue[T any](n int, in <-chan T) <-chan T {
	return Buffer(n, in)
}

// WithContext adds context cancellation to a pipeline stage
func WithContext[T any](ctx context.Context, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()

	return out
}

// Context is an alias for WithContext for backward compatibility
func Context[T any](ctx context.Context, in <-chan T) <-chan T {
	return WithContext(ctx, in)
}

// WithTimeout adds timeout to a pipeline stage
func WithTimeout[T any](timeout time.Duration, in <-chan T) <-chan T {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	out := make(chan T)
	go func() {
		defer cancel()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				// Drain the input channel to prevent goroutine leaks
				go func() {
					for range in {
					}
				}()
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()

	return out
}

// OrDone provides done channel cancellation
func OrDone[T any](done <-chan struct{}, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case out <- v:
				}
			}
		}
	}()

	return out
}

// PassThrough passes values through while executing a side effect
func PassThrough[T any](in <-chan T, fn func(T)) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			fn(v)
			out <- v
		}
	}()

	return out
}

// Tap is an alias for PassThrough
func Tap[T any](in <-chan T, fn func(T)) <-chan T {
	return PassThrough(in, fn)
}

// Monitor adds monitoring to a pipeline stage
func Monitor[T any](in <-chan T, onItem func(T), onError func(error)) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			if onItem != nil {
				withRecovery(func() { onItem(v) }, func(r interface{}) {
					if onError != nil {
						onError(fmt.Errorf("monitor panic: %v", r))
					}
				})
			}
			out <- v
		}
	}()

	return out
}

// Throughput monitors throughput and error rates
func Throughput[T any](in <-chan Result[T], fn func(ThroughputInfo)) <-chan Result[T] {
	out := make(chan Result[T])

	var total, totalFailures int64
	startTime := time.Now()
	er := rate.NewErrors(time.Second)

	go func() {
		defer close(out)

		for v := range in {
			_, err := v.Unwrap()
			if err != nil {
				atomic.AddInt64(&totalFailures, 1)
				er.Failure().Inc()
			} else {
				er.Success().Inc()
			}
			atomic.AddInt64(&total, 1)

			if fn != nil {
				r := er.Rate()
				fn(ThroughputInfo{
					Total:         int(atomic.LoadInt64(&total)),
					TotalFailures: int(atomic.LoadInt64(&totalFailures)),
					ErrorRate:     r.Ratio(),
					Rate:          r.Total(),
					Duration:      time.Since(startTime),
				})
			}
			out <- v
		}
	}()

	return out
}

// Rate monitors processing rate
func Rate[T any](in <-chan T, fn func(RateInfo)) <-chan T {
	out := make(chan T)

	var total int64
	r := rate.NewRate(time.Second)

	go func() {
		defer close(out)

		for v := range in {
			atomic.AddInt64(&total, 1)
			r.Inc()

			if fn != nil {
				fn(RateInfo{
					Total: int(atomic.LoadInt64(&total)),
					Rate:  r.Count(),
				})
			}
			out <- v
		}
	}()

	return out
}

// Transform applies a transformation function to each item
func Transform[T, V any](in <-chan T, fn func(T) V) <-chan V {
	out := make(chan V)

	go func() {
		defer close(out)

		for v := range in {
			out <- fn(v)
		}
	}()

	return out
}

// Map is an alias for Transform for backward compatibility
func Map[T, V any](in <-chan T, fn func(T) V) <-chan V {
	return Transform(in, fn)
}

// Pipe applies a transformation that returns the same type
func Pipe[T any](in <-chan T, fn func(T) T) <-chan T {
	return Transform(in, fn)
}

// Filter keeps only items that match the predicate
func Filter[T any](in <-chan T, predicate func(T) bool) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			if predicate(v) {
				out <- v
			}
		}
	}()

	return out
}

// Take takes the first n items from the channel
func Take[T any](n int, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		count := 0
		for v := range in {
			if count >= n {
				break
			}
			out <- v
			count++
		}
	}()

	return out
}

// Skip skips the first n items from the channel
func Skip[T any](n int, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		count := 0
		for v := range in {
			if count < n {
				count++
				continue
			}
			out <- v
		}
	}()

	return out
}

// Distinct removes duplicate items (requires comparable type)
func Distinct[T comparable](in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		seen := make(map[T]struct{})
		for v := range in {
			if _, exists := seen[v]; !exists {
				seen[v] = struct{}{}
				out <- v
			}
		}
	}()

	return out
}

// Pool runs n workers in parallel to process items
func Pool[T, V any](n int, in <-chan T, fn func(T) V) <-chan V {
	if n <= 0 {
		panic(ErrInvalidWorkers)
	}

	out := make(chan V)

	var wg sync.WaitGroup
	wg.Add(n)

	// Use buffered channel to prevent blocking when workers complete at different times
	results := make(chan V, n)

	for i := 0; i < n; i++ {
		go func(workerID int) {
			defer wg.Done()

			for v := range in {
				result := fn(v)
				results <- result
			}
		}(i)
	}

	// Goroutine to forward results and handle cleanup
	go func() {
		wg.Wait()
		close(results)
	}()

	// Forward results from buffered channel to output
	go func() {
		defer close(out)
		for result := range results {
			out <- result
		}
	}()

	return out
}

// PoolWithContext runs n workers with context cancellation
func PoolWithContext[T, V any](ctx context.Context, n int, in <-chan T, fn func(context.Context, T) V) <-chan V {
	if n <= 0 {
		panic(ErrInvalidWorkers)
	}

	out := make(chan V)

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-in:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case out <- fn(ctx, v):
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// FanOut distributes input to multiple output channels
func FanOut[T any](n int, in <-chan T) []<-chan T {
	if n <= 0 {
		panic(ErrInvalidWorkers)
	}

	outputs := make([]chan T, n)
	for i := range outputs {
		outputs[i] = make(chan T)
	}

	// Convert to read-only channels
	readOnlyOutputs := make([]<-chan T, n)
	for i, ch := range outputs {
		readOnlyOutputs[i] = ch
	}

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()

		i := 0
		for v := range in {
			outputs[i] <- v
			i = (i + 1) % n
		}
	}()

	return readOnlyOutputs
}

// FanIn merges multiple input channels into one output channel
func FanIn[T any](channels ...<-chan T) <-chan T {
	out := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(c <-chan T) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Semaphore limits concurrent execution using a semaphore
func Semaphore[T, V any](n int, in <-chan T, fn func(T) V) <-chan V {
	if n <= 0 {
		panic(ErrInvalidWorkers)
	}

	out := make(chan V)
	ctx := context.Background()
	sem := semaphore.NewWeighted(int64(n))

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		for v := range in {
			wg.Add(1)

			go func(item T) {
				defer wg.Done()

				_ = sem.Acquire(ctx, 1)
				defer sem.Release(1)

				out <- fn(item)
			}(v)
		}
		wg.Wait()
	}()

	return out
}

// Throttle limits the rate of items passing through
func Throttle[T any](interval time.Duration, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for v := range in {
			<-ticker.C
			out <- v
		}
	}()

	return out
}

// Every processes items at regular intervals
func Every[T any](interval time.Duration, in <-chan T) <-chan T {
	return Throttle(interval, in)
}

// RateLimit limits the rate of processing
func RateLimit[T any](requestsPerSecond int, in <-chan T) <-chan T {
	if requestsPerSecond <= 0 {
		panic(ErrInvalidWorkers)
	}

	interval := time.Second / time.Duration(requestsPerSecond)
	return Throttle(interval, in)
}

// Debounce ensures minimum time between emissions
func Debounce[T any](duration time.Duration, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		var lastEmit time.Time
		for v := range in {
			if time.Since(lastEmit) >= duration {
				out <- v
				lastEmit = time.Now()
			}
		}
	}()

	return out
}

// Batch collects items into batches
func Batch[T any](size int, timeout time.Duration, in <-chan T) <-chan []T {
	if size <= 0 {
		panic(ErrInvalidBufferSize)
	}

	out := make(chan []T)

	go func() {
		defer close(out)

		var batch []T
		timer := time.NewTimer(timeout)
		timer.Stop()

		flush := func() {
			if len(batch) > 0 {
				out <- batch
				batch = nil
			}
		}

		for {
			select {
			case v, ok := <-in:
				if !ok {
					flush()
					return
				}

				batch = append(batch, v)
				if len(batch) == 1 {
					timer.Reset(timeout)
				}

				if len(batch) >= size {
					timer.Stop()
					flush()
				}

			case <-timer.C:
				flush()
			}
		}
	}()

	return out
}

// BatchDistinct collects unique items into batches
func BatchDistinct[T comparable](size int, timeout time.Duration, in <-chan T) <-chan []T {
	if size <= 0 {
		panic(ErrInvalidBufferSize)
	}

	out := make(chan []T)

	go func() {
		defer close(out)

		cache := make(map[T]struct{})
		timer := time.NewTimer(timeout)
		timer.Stop()

		flush := func() {
			if len(cache) > 0 {
				batch := make([]T, 0, len(cache))
				for k := range cache {
					batch = append(batch, k)
				}
				clear(cache)
				out <- batch
			}
		}

		for {
			select {
			case v, ok := <-in:
				if !ok {
					flush()
					return
				}

				if _, exists := cache[v]; !exists {
					cache[v] = struct{}{}
					if len(cache) == 1 {
						timer.Reset(timeout)
					}

					if len(cache) >= size {
						timer.Stop()
						flush()
					}
				}

			case <-timer.C:
				flush()
			}
		}
	}()

	return out
}

// Tee splits a channel into two identical channels
func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	out1, out2 := make(chan T), make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for v := range in {
			// Send to both channels, blocking until both are ready
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				out1 <- v
			}()

			go func() {
				defer wg.Done()
				out2 <- v
			}()

			wg.Wait()
		}
	}()

	return out1, out2
}

// FlatMap flattens results, keeping only successful ones
func FlatMap[T any](in <-chan Result[T]) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for result := range in {
			if result.IsSuccess() {
				out <- result.Data
			}
		}
	}()

	return out
}

// FilterErrors filters out errors and calls the error handler
func FilterErrors[T any](in <-chan Result[T], onError func(error)) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for result := range in {
			if result.IsError() {
				if onError != nil {
					onError(result.Err)
				}
			} else {
				out <- result.Data
			}
		}
	}()

	return out
}

// Error is an alias for FilterErrors for backward compatibility
func Error[T any](in <-chan Result[T], onError func(error)) <-chan T {
	return FilterErrors(in, onError)
}

// Retry retries failed operations
func Retry[T any](maxRetries int, backoff time.Duration, in <-chan Result[T], retryFn func(T) Result[T]) <-chan Result[T] {
	out := make(chan Result[T])

	go func() {
		defer close(out)

		for result := range in {
			if result.IsSuccess() {
				out <- result
				continue
			}

			// Retry logic
			var lastResult Result[T] = result
			for i := 0; i < maxRetries; i++ {
				time.Sleep(backoff)
				lastResult = retryFn(result.Data)
				if lastResult.IsSuccess() {
					break
				}
			}
			out <- lastResult
		}
	}()

	return out
}

// Timeout adds timeout to individual operations
func Timeout[T any](duration time.Duration, in <-chan T) <-chan Result[T] {
	out := make(chan Result[T])

	go func() {
		defer close(out)

		for v := range in {
			select {
			case out <- MakeSuccessResult(v):
			case <-time.After(duration):
				out <- MakeErrorResult[T](ErrInvalidTimeout)
			}
		}
	}()

	return out
}

// First returns the first item from the channel
func First[T any](in <-chan T) (T, bool) {
	v, ok := <-in
	return v, ok
}

// Last returns the last item from the channel
func Last[T any](in <-chan T) (T, bool) {
	var last T
	var hasValue bool

	for v := range in {
		last = v
		hasValue = true
	}

	return last, hasValue
}

// ToSlice collects all items into a slice
func ToSlice[T any](in <-chan T) []T {
	var result []T
	for v := range in {
		result = append(result, v)
	}
	return result
}

// ForEach processes each item with a function
func ForEach[T any](in <-chan T, fn func(T)) {
	for v := range in {
		fn(v)
	}
}

// Merge merges multiple channels using a merge function
func Merge[T any](mergeFn func(T, T) T, channels ...<-chan T) <-chan T {
	if len(channels) == 0 {
		out := make(chan T)
		close(out)
		return out
	}

	if len(channels) == 1 {
		return channels[0]
	}

	// Binary merge
	var merge func(left, right <-chan T) <-chan T
	merge = func(left, right <-chan T) <-chan T {
		out := make(chan T)

		go func() {
			defer close(out)

			for {
				select {
				case v1, ok1 := <-left:
					if !ok1 {
						// Left channel closed, drain right
						for v2 := range right {
							out <- v2
						}
						return
					}

					select {
					case v2, ok2 := <-right:
						if !ok2 {
							// Right channel closed, send left value and drain left
							out <- v1
							for v := range left {
								out <- v
							}
							return
						}
						out <- mergeFn(v1, v2)
					case out <- v1:
						// No value from right, send left value
					}

				case v2, ok2 := <-right:
					if !ok2 {
						// Right channel closed, drain left
						for v1 := range left {
							out <- v1
						}
						return
					}
					out <- v2
				}
			}
		}()

		return out
	}

	// Merge all channels in pairs
	result := channels[0]
	for i := 1; i < len(channels); i++ {
		result = merge(result, channels[i])
	}

	return result
}

// PipelineMetricsCollector defines the interface for collecting pipeline metrics.
type PipelineMetricsCollector interface {
	IncProcessedCount()
	IncErrorCount()
	IncPanicCount()
	SetStartTime(t time.Time)
	SetDuration(d time.Duration)
	SetThroughputRate(rate float64)
	SetErrorRate(rate float64)
	GetMetrics() Metrics
}

// AtomicPipelineMetricsCollector is the default atomic-based metrics implementation.
type AtomicPipelineMetricsCollector struct {
	processedCount int64
	errorCount     int64
	panicCount     int64
	startTime      atomic.Value // time.Time
	duration       int64        // nanoseconds
	throughputRate atomic.Value // float64
	errorRate      atomic.Value // float64
}

func (m *AtomicPipelineMetricsCollector) IncProcessedCount() { atomic.AddInt64(&m.processedCount, 1) }
func (m *AtomicPipelineMetricsCollector) IncErrorCount()     { atomic.AddInt64(&m.errorCount, 1) }
func (m *AtomicPipelineMetricsCollector) IncPanicCount()     { atomic.AddInt64(&m.panicCount, 1) }
func (m *AtomicPipelineMetricsCollector) SetStartTime(t time.Time) { m.startTime.Store(t) }
func (m *AtomicPipelineMetricsCollector) SetDuration(d time.Duration) { atomic.StoreInt64(&m.duration, int64(d)) }
func (m *AtomicPipelineMetricsCollector) SetThroughputRate(rate float64) { m.throughputRate.Store(rate) }
func (m *AtomicPipelineMetricsCollector) SetErrorRate(rate float64)      { m.errorRate.Store(rate) }
func (m *AtomicPipelineMetricsCollector) GetMetrics() Metrics {
	var startTime time.Time
	if v := m.startTime.Load(); v != nil {
		startTime = v.(time.Time)
	}
	var throughputRate, errorRate float64
	if v := m.throughputRate.Load(); v != nil {
		throughputRate = v.(float64)
	}
	if v := m.errorRate.Load(); v != nil {
		errorRate = v.(float64)
	}
	return Metrics{
		ProcessedCount: atomic.LoadInt64(&m.processedCount),
		ErrorCount:     atomic.LoadInt64(&m.errorCount),
		PanicCount:     atomic.LoadInt64(&m.panicCount),
		StartTime:      startTime,
		Duration:       time.Duration(atomic.LoadInt64(&m.duration)),
		ThroughputRate: throughputRate,
		ErrorRate:      errorRate,
	}
}

// PrometheusPipelineMetricsCollector implements PipelineMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusPipelineMetricsCollector struct {
	ProcessedCount prometheus.Counter
	ErrorCount     prometheus.Counter
	PanicCount     prometheus.Counter
	StartTime      prometheus.Gauge
	Duration       prometheus.Gauge
	ThroughputRate prometheus.Gauge
	ErrorRate      prometheus.Gauge
}

func (m *PrometheusPipelineMetricsCollector) IncProcessedCount() { m.ProcessedCount.Inc() }
func (m *PrometheusPipelineMetricsCollector) IncErrorCount()     { m.ErrorCount.Inc() }
func (m *PrometheusPipelineMetricsCollector) IncPanicCount()     { m.PanicCount.Inc() }
func (m *PrometheusPipelineMetricsCollector) SetStartTime(t time.Time) { m.StartTime.Set(float64(t.Unix())) }
func (m *PrometheusPipelineMetricsCollector) SetDuration(d time.Duration) { m.Duration.Set(float64(d.Seconds())) }
func (m *PrometheusPipelineMetricsCollector) SetThroughputRate(rate float64) { m.ThroughputRate.Set(rate) }
func (m *PrometheusPipelineMetricsCollector) SetErrorRate(rate float64)      { m.ErrorRate.Set(rate) }
func (m *PrometheusPipelineMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}
