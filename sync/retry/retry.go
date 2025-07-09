// Package retry implements retry mechanism with throttler.
package retry

import (
	"context"
	"errors"
	"iter"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrThrottled     = errors.New("retry: throttled")
)

type retry interface {
	Try(ctx context.Context, limit int) iter.Seq2[int, error]
}

var _ retry = (*Retry)(nil)

type Retry struct {
	BackOff          backOff
	Throttler        throttler
	metricsCollector RetryMetricsCollector
}

func New() *Retry {
	var t *Throttler

	return &Retry{
		BackOff:          NewExponentialBackOff(time.Second, time.Minute),
		Throttler:        t,
		metricsCollector: &AtomicRetryMetricsCollector{},
	}
}

func (r *Retry) WithBackOff(policy backOff) *Retry {
	r.BackOff = policy
	return r
}

func (r *Retry) WithMetricsCollector(collector RetryMetricsCollector) *Retry {
	if collector != nil {
		r.metricsCollector = collector
	}
	return r
}

func (r *Retry) Try(ctx context.Context, limit int) iter.Seq2[int, error] {
	return func(yield func(int, error) bool) {
		for i := range limit + 1 {
			r.metricsCollector.IncAttempts()
			if i == limit {
				r.metricsCollector.IncLimitExceeded()
				yield(i, ErrLimitExceeded)
				break
			}

			// Throttle only applies to retries, skip the first call.
			if i > 0 && !r.Throttler.Allow() {
				r.metricsCollector.IncThrottles()
				yield(i, ErrThrottled)
				break
			}

			if err := ctx.Err(); err != nil {
				r.metricsCollector.IncFailures()
				yield(i, err)
				return
			}

			if !yield(i, nil) {
				// Breaking early is considered a success.
				r.metricsCollector.IncSuccesses()
				r.Throttler.Success()
				break
			}

			// Using time.Sleep blocks the operation and cannot be cancelled in case
			// timeout becomes very long.
			// Use time.After combined with context instead.
			select {
			case <-ctx.Done():
			case <-time.After(r.BackOff.At(i)):
			}
		}
	}
}

func (r *Retry) Do(ctx context.Context, fn func(context.Context) error, limit int) (err error) {
	for _, retryErr := range r.Try(ctx, limit) {
		if retryErr != nil {
			r.metricsCollector.IncFailures()
			return errors.Join(retryErr, err)
		}

		err = fn(ctx)
		if err == nil {
			r.metricsCollector.IncSuccesses()
			break
		} else {
			r.metricsCollector.IncFailures()
		}
	}

	return nil
}

// RetryMetricsCollector defines the interface for collecting retry metrics.
type RetryMetricsCollector interface {
	IncAttempts()
	IncSuccesses()
	IncFailures()
	IncThrottles()
	IncLimitExceeded()
	GetMetrics() RetryMetrics
}

type RetryMetrics struct {
	Attempts      int64
	Successes     int64
	Failures      int64
	Throttles     int64
	LimitExceeded int64
}

// AtomicRetryMetricsCollector is the default atomic-based metrics implementation.
type AtomicRetryMetricsCollector struct {
	attempts      int64
	successes     int64
	failures      int64
	throttles     int64
	limitExceeded int64
}

func (m *AtomicRetryMetricsCollector) IncAttempts()      { atomic.AddInt64(&m.attempts, 1) }
func (m *AtomicRetryMetricsCollector) IncSuccesses()     { atomic.AddInt64(&m.successes, 1) }
func (m *AtomicRetryMetricsCollector) IncFailures()      { atomic.AddInt64(&m.failures, 1) }
func (m *AtomicRetryMetricsCollector) IncThrottles()     { atomic.AddInt64(&m.throttles, 1) }
func (m *AtomicRetryMetricsCollector) IncLimitExceeded() { atomic.AddInt64(&m.limitExceeded, 1) }
func (m *AtomicRetryMetricsCollector) GetMetrics() RetryMetrics {
	return RetryMetrics{
		Attempts:      atomic.LoadInt64(&m.attempts),
		Successes:     atomic.LoadInt64(&m.successes),
		Failures:      atomic.LoadInt64(&m.failures),
		Throttles:     atomic.LoadInt64(&m.throttles),
		LimitExceeded: atomic.LoadInt64(&m.limitExceeded),
	}
}

// PrometheusRetryMetricsCollector implements RetryMetricsCollector using prometheus metrics.
type PrometheusRetryMetricsCollector struct {
	Attempts      prometheus.Counter
	Successes     prometheus.Counter
	Failures      prometheus.Counter
	Throttles     prometheus.Counter
	LimitExceeded prometheus.Counter
}

func (m *PrometheusRetryMetricsCollector) IncAttempts()      { m.Attempts.Inc() }
func (m *PrometheusRetryMetricsCollector) IncSuccesses()     { m.Successes.Inc() }
func (m *PrometheusRetryMetricsCollector) IncFailures()      { m.Failures.Inc() }
func (m *PrometheusRetryMetricsCollector) IncThrottles()     { m.Throttles.Inc() }
func (m *PrometheusRetryMetricsCollector) IncLimitExceeded() { m.LimitExceeded.Inc() }
func (m *PrometheusRetryMetricsCollector) GetMetrics() RetryMetrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return RetryMetrics{}
}
