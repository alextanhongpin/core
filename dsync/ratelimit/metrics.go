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
}

type AtomicMetricsCollector struct {
	totalRequests int64
	allowed       int64
	denied        int64
}

func (m *AtomicMetricsCollector) IncTotalRequests() { atomic.AddInt64(&m.totalRequests, 1) }
func (m *AtomicMetricsCollector) IncAllowed()       { atomic.AddInt64(&m.allowed, 1) }
func (m *AtomicMetricsCollector) IncDenied()        { atomic.AddInt64(&m.denied, 1) }

// PrometheusMetricsCollector implements MetricsCollector using prometheus metrics.
type PrometheusMetricsCollector struct {
	TotalRequests prometheus.Counter
	Allowed       prometheus.Counter
	Denied        prometheus.Counter
}

func (m *PrometheusMetricsCollector) IncTotalRequests() { m.TotalRequests.Inc() }
func (m *PrometheusMetricsCollector) IncAllowed()       { m.Allowed.Inc() }
func (m *PrometheusMetricsCollector) IncDenied()        { m.Denied.Inc() }
