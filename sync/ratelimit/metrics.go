package ratelimit

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector defines the interface for collecting rate limiter metrics.
type MetricsCollector interface {
	IncTotalRequests()
	IncAllowed()
	IncDenied()
	GetMetrics() Metrics
}

type Metrics struct {
	TotalRequests int64
	Allowed       int64
	Denied        int64
}

// AtomicMetricsCollector is the default atomic-based metrics implementation.
type AtomicMetricsCollector struct {
	totalRequests int64
	allowed       int64
	denied        int64
}

func (m *AtomicMetricsCollector) IncTotalRequests() { atomic.AddInt64(&m.totalRequests, 1) }
func (m *AtomicMetricsCollector) IncAllowed()       { atomic.AddInt64(&m.allowed, 1) }
func (m *AtomicMetricsCollector) IncDenied()        { atomic.AddInt64(&m.denied, 1) }
func (m *AtomicMetricsCollector) GetMetrics() Metrics {
	return Metrics{
		TotalRequests: atomic.LoadInt64(&m.totalRequests),
		Allowed:       atomic.LoadInt64(&m.allowed),
		Denied:        atomic.LoadInt64(&m.denied),
	}
}

// PrometheusMetricsCollector implements MetricsCollector using prometheus metrics.
type PrometheusMetricsCollector struct {
	TotalRequests prometheus.Counter
	Allowed       prometheus.Counter
	Denied        prometheus.Counter
}

func (m *PrometheusMetricsCollector) IncTotalRequests() { m.TotalRequests.Inc() }
func (m *PrometheusMetricsCollector) IncAllowed()       { m.Allowed.Inc() }
func (m *PrometheusMetricsCollector) IncDenied()        { m.Denied.Inc() }
func (m *PrometheusMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}
