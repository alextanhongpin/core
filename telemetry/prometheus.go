// This is a modified version of https://pkg.go.dev/golang.org/x/exp/event@v0.0.0-20230817173708-d852ddb80c63/otel, since the supported OTEL package is no longer the latest.
package telemetry

import (
	"context"
	"errors"
	"fmt"
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
	recordFuncs map[event.Metric]recordFunc
	collectors  map[string]prometheus.Collector
}

var _ event.Handler = (*PrometheusHandler)(nil)

// NewPrometheusHandler creates a new PrometheusHandler.
func NewPrometheusHandler(client prometheus.Registerer) *PrometheusHandler {
	return &PrometheusHandler{
		client:      client,
		recordFuncs: map[event.Metric]recordFunc{},
		collectors:  make(map[string]prometheus.Collector),
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

	rf := m.getRecordFunc(em, e.Labels)
	if rf == nil {
		panic(fmt.Errorf("unable to record for metric %v", em))
	}
	rf(ctx, lval, e.Labels)
	return ctx
}

func (m *PrometheusHandler) Collector(name string) prometheus.Collector {
	return m.collectors[name]
}

func (m *PrometheusHandler) getRecordFunc(em event.Metric, labels []event.Label) recordFunc {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.recordFuncs[em]; ok {
		return f
	}
	f := m.newRecordFunc(em, labels)
	m.recordFuncs[em] = f
	return f
}

func (m *PrometheusHandler) newRecordFunc(em event.Metric, labels []event.Label) recordFunc {
	opts := em.Options()
	name := em.Name()

	switch opts.Unit {
	case event.UnitDimensionless:
	case event.UnitBytes:
		name += "_bytes"
	case event.UnitMilliseconds:
		name += "_milliseconds"
	}

	keys, _ := labelsToKeyVals(labels)

	switch em.(type) {
	case *event.Counter:
		c := prometheus.NewCounterVec(prometheus.CounterOpts{
			// NOTE: This will use the github package name, which panics on registering.
			//Namespace: opts.Namespace,
			Name: name,
			Help: opts.Description,
		}, keys)
		m.collectors[name] = c
		m.client.MustRegister(c)

		return func(ctx context.Context, l event.Label, labels []event.Label) {
			_, vals := labelsToKeyVals(labels)
			c.WithLabelValues(vals...).Add(float64(l.Int64()))
		}

	case *event.FloatGauge:
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			// NOTE: This will use the github package name, which panics on registering.
			//Namespace: opts.Namespace,
			Name: name,
			Help: opts.Description,
		}, keys)
		m.client.MustRegister(g)

		return func(ctx context.Context, l event.Label, labels []event.Label) {
			_, vals := labelsToKeyVals(labels)
			g.WithLabelValues(vals...).Add(l.Float64())
		}
	case *event.DurationDistribution:
		r := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			// NOTE: This will use the github package name, which panics on registering.
			//Namespace: opts.Namespace,
			Name: name,
			Help: opts.Description,
		}, keys)
		m.client.MustRegister(r)

		return func(ctx context.Context, l event.Label, attrs []event.Label) {
			_, vals := labelsToKeyVals(labels)
			r.WithLabelValues(vals...).Observe(float64(l.Duration().Nanoseconds()))
		}
	default:
		return nil
	}
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
