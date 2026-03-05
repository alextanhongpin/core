package retry

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

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
