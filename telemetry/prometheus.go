// This is a modified version of https://pkg.go.dev/golang.org/x/exp/event@v0.0.0-20230817173708-d852ddb80c63/otel, since the supported OTEL package is no longer the latest.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/event"
)

var (
	ErrNilRegisterer                   = errors.New("prometheus registerer cannot be nil")
	ErrUnsupportedPrometheusMetricType = errors.New("unsupported metric type for PrometheusHandler")
	ErrUnsupportedCollectorType        = errors.New("unsupported collector type for PrometheusHandler")
)

// PrometheusHandler is an event.Handler for Prometheus metrics.
// Its Event method handles Metric events and ignores all others.
type PrometheusHandler struct {
	client prometheus.Registerer
	mu     sync.RWMutex
	// A map from event.Metrics to prometheus collectors.
	collectors map[string]prometheus.Collector
	// errorHandler allows custom error handling instead of panicking
	errorHandler func(error)
}

var _ event.Handler = (*PrometheusHandler)(nil)

// PrometheusHandlerOption configures a PrometheusHandler.
type PrometheusHandlerOption func(*PrometheusHandler)

// WithPrometheusErrorHandler sets a custom error handler for the PrometheusHandler.
// If not set, errors will be logged using the default logger.
func WithPrometheusErrorHandler(handler func(error)) PrometheusHandlerOption {
	return func(p *PrometheusHandler) {
		p.errorHandler = handler
	}
}

// NewPrometheusHandler creates a new PrometheusHandler.
func NewPrometheusHandler(client prometheus.Registerer, opts ...PrometheusHandlerOption) (*PrometheusHandler, error) {
	if client == nil {
		return nil, ErrNilRegisterer
	}

	handler := &PrometheusHandler{
		client:     client,
		collectors: make(map[string]prometheus.Collector),
		errorHandler: func(err error) {
			log.Printf("PrometheusHandler error: %v", err)
		},
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

func (m *PrometheusHandler) Event(ctx context.Context, e *event.Event) context.Context {
	if e == nil {
		m.handleError(ErrNilEvent)
		return ctx
	}

	if e.Kind != event.MetricKind {
		return ctx
	}

	// Get the prometheus instrument corresponding to the event's MetricDescriptor,
	// or create a new one.
	mi, ok := event.MetricKey.Find(e)
	if !ok {
		m.handleError(ErrNoMetricKey)
		return ctx
	}

	em, ok := mi.(event.Metric)
	if !ok {
		m.handleError(fmt.Errorf("metric key is not of type event.Metric: %T", mi))
		return ctx
	}

	lval := e.Find(event.MetricVal)
	if !lval.HasValue() {
		m.handleError(ErrNoMetricValue)
		return ctx
	}

	name := em.Name()
	if name == "" {
		m.handleError(errors.New("metric name cannot be empty"))
		return ctx
	}

	opts := em.Options()

	nameWithUnit := name
	switch opts.Unit {
	case event.UnitDimensionless:
	case event.UnitBytes:
		nameWithUnit += "_bytes"
	}

	keys, vals := labelsToKeyVals(e.Labels)

	if err := m.ensureCollector(em, nameWithUnit, &opts, keys); err != nil {
		m.handleError(fmt.Errorf("failed to ensure collector for %s: %w", name, err))
		return ctx
	}

	if err := m.recordMetric(name, nameWithUnit, &opts, lval, vals); err != nil {
		m.handleError(fmt.Errorf("failed to record metric %s: %w", name, err))
	}

	return ctx
}

func (m *PrometheusHandler) ensureCollector(em event.Metric, nameWithUnit string, opts *event.MetricOptions, keys []string) error {
	name := em.Name()

	// Use read lock first to check if collector exists
	m.mu.RLock()
	_, exists := m.collectors[name]
	m.mu.RUnlock()

	if exists {
		return nil
	}

	// Upgrade to write lock to create collector
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check pattern
	if _, exists := m.collectors[name]; exists {
		return nil
	}

	var c prometheus.Collector

	switch em.(type) {
	case *event.Counter:
		c = prometheus.NewCounterVec(prometheus.CounterOpts{
			Help:      opts.Description,
			Name:      nameWithUnit,
			Namespace: opts.Namespace,
		}, keys)
	case *event.FloatGauge:
		c = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Help:      opts.Description,
			Name:      nameWithUnit,
			Namespace: opts.Namespace,
		}, keys)
	case *event.DurationDistribution:
		histogramName := nameWithUnit
		switch opts.Unit {
		case event.UnitMilliseconds:
			histogramName += "_milliseconds"
		default:
			histogramName += "_seconds"
		}
		c = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Help:      opts.Description,
			Name:      histogramName,
			Namespace: opts.Namespace,
		}, keys)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedPrometheusMetricType, name)
	}

	m.collectors[name] = c

	// Handle registration errors gracefully
	if err := m.client.Register(c); err != nil {
		// Check if it's already registered error
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// Use the existing collector
			m.collectors[name] = are.ExistingCollector
		} else {
			return fmt.Errorf("failed to register collector %s: %w", name, err)
		}
	}

	return nil
}

func (m *PrometheusHandler) recordMetric(name, nameWithUnit string, opts *event.MetricOptions, lval event.Label, vals []string) error {
	m.mu.RLock()
	c, ok := m.collectors[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("collector not found for metric %s", name)
	}

	switch col := c.(type) {
	case *prometheus.CounterVec:
		value := float64(lval.Int64())
		if value < 0 {
			return fmt.Errorf("counter value cannot be negative: %f", value)
		}
		col.WithLabelValues(vals...).Add(value)
	case *prometheus.GaugeVec:
		col.WithLabelValues(vals...).Set(lval.Float64())
	case *prometheus.HistogramVec:
		duration := lval.Duration()
		if duration < 0 {
			return fmt.Errorf("duration cannot be negative: %v", duration)
		}

		durationValue := duration.Seconds()
		if opts.Unit == event.UnitMilliseconds {
			durationValue = float64(duration.Milliseconds())
		}
		col.WithLabelValues(vals...).Observe(durationValue)
	default:
		return fmt.Errorf("%w: %s (type: %T)", ErrUnsupportedCollectorType, name, col)
	}

	return nil
}

func (m *PrometheusHandler) handleError(err error) {
	if m.errorHandler != nil {
		m.errorHandler(err)
	}
}

// Collector returns the prometheus collector for the given metric name and whether it exists.
func (m *PrometheusHandler) Collector(name string) (prometheus.Collector, bool) {
	if name == "" {
		return nil, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.collectors[name]
	return c, ok
}

// Close cleans up resources used by the PrometheusHandler.
// It's safe to call multiple times.
func (m *PrometheusHandler) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister all collectors
	for name, collector := range m.collectors {
		if !m.client.Unregister(collector) {
			// Log but don't fail if unregistering fails
			m.handleError(fmt.Errorf("failed to unregister collector %s", name))
		}
	}

	// Clear the map to prevent memory leaks
	m.collectors = make(map[string]prometheus.Collector)
	return nil
}

func labelsToKeyVals(labels []event.Label) (keys []string, vals []string) {
	for _, l := range labels {
		if l.Name == string(event.MetricKey) || l.Name == string(event.MetricVal) {
			continue
		}
		// Skip empty labels
		if l.Name == "" || !l.HasValue() {
			continue
		}
		keys = append(keys, l.Name)
		vals = append(vals, l.String())
	}

	return
}
