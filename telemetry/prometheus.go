// This is a modified version of https://pkg.go.dev/golang.org/x/exp/event@v0.0.0-20230817173708-d852ddb80c63/otel, since the supported OTEL package is no longer the latest.
package telemetry

import (
	"context"
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/event"
)

// PrometheusHandler is an event.Handler for OpenTelemetry metrics.
// Its Event method handles Metric events and ignores all others.
type PrometheusHandler struct {
	client prometheus.Registerer
	mu     sync.Mutex
	// A map from event.Metrics to, effectively, otel Meters.
	// But since the only thing we need from the Meter is recording a value, we
	// use a function for that that closes over the Meter itself.
	collectors map[string]prometheus.Collector
}

var _ event.Handler = (*PrometheusHandler)(nil)

// NewPrometheusHandler creates a new PrometheusHandler.
func NewPrometheusHandler(client prometheus.Registerer) *PrometheusHandler {
	return &PrometheusHandler{
		client:     client,
		collectors: make(map[string]prometheus.Collector),
	}
}

func (m *PrometheusHandler) Event(ctx context.Context, e *event.Event) context.Context {
	if e.Kind != event.MetricKind {
		return ctx
	}
	// Get the otel instrument corresponding to the event's MetricDescriptor,
	// or create a new one.
	mi, ok := event.MetricKey.Find(e)
	if !ok {
		panic(errors.New("no metric key for metric event"))
	}

	em := mi.(event.Metric)
	lval := e.Find(event.MetricVal)
	if !lval.HasValue() {
		panic(errors.New("no metric value for metric event"))
	}

	name := em.Name()
	opts := em.Options()

	nameWithUnit := name
	switch opts.Unit {
	case event.UnitDimensionless:
	case event.UnitBytes:
		nameWithUnit += "_bytes"
	}

	keys, vals := labelsToKeyVals(e.Labels)

	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.collectors[name]
	if !ok {
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
			switch opts.Unit {
			case event.UnitMilliseconds:
				nameWithUnit += "_milliseconds"
			default:
				nameWithUnit += "_seconds"
			}
			c = prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Help:      opts.Description,
				Name:      nameWithUnit,
				Namespace: opts.Namespace,
			}, keys)
		default:
			panic(errors.New("unsupported metric type for PrometheusHandler: " + em.Name()))
		}
		m.collectors[name] = c
		m.client.MustRegister(c)
	}

	switch col := c.(type) {
	case *prometheus.CounterVec:
		col.WithLabelValues(vals...).Add(float64(lval.Int64()))
	case *prometheus.GaugeVec:
		col.WithLabelValues(vals...).Set(lval.Float64())
	case *prometheus.HistogramVec:
		duration := lval.Duration().Seconds()
		if opts.Unit == event.UnitMilliseconds {
			duration = float64(lval.Duration().Milliseconds())
		}
		col.WithLabelValues(vals...).Observe(duration)
	default:
		panic(errors.New("unsupported collector type for PrometheusHandler: " + em.Name()))
	}
	return ctx
}

func (m *PrometheusHandler) Collector(name string) (prometheus.Collector, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.collectors[name]
	return c, ok
}

func labelsToKeyVals(labels []event.Label) (keys []string, vals []string) {
	for _, l := range labels {
		if l.Name == string(event.MetricKey) || l.Name == string(event.MetricVal) {
			continue
		}
		keys = append(keys, l.Name)
		vals = append(vals, l.String())
	}

	return
}
