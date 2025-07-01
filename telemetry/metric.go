// This is a modified version of https://pkg.go.dev/golang.org/x/exp/event@v0.0.0-20230817173708-d852ddb80c63/otel, since the supported OTEL package is no longer the latest.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/event"
)

var (
	ErrNoMetricKey           = errors.New("no metric key for metric event")
	ErrNoMetricValue         = errors.New("no metric value for metric event")
	ErrNilMeter              = errors.New("meter cannot be nil")
	ErrNilEvent              = errors.New("event cannot be nil")
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
)

// MetricHandler is an event.Handler for OpenTelemetry metrics.
// Its Event method handles Metric events and ignores all others.
type MetricHandler struct {
	meter metric.Meter
	mu    sync.RWMutex
	// A map from event.Metrics to, effectively, otel Meters.
	// But since the only thing we need from the Meter is recording a value, we
	// use a function for that that closes over the Meter itself.
	recordFuncs map[event.Metric]recordFunc
	// errorHandler allows custom error handling instead of panicking
	errorHandler func(error)
}

type recordFunc func(context.Context, event.Label, []event.Label) error

var _ event.Handler = (*MetricHandler)(nil)

// MetricHandlerOption configures a MetricHandler.
type MetricHandlerOption func(*MetricHandler)

// WithErrorHandler sets a custom error handler for the MetricHandler.
// If not set, errors will be logged using the default logger.
func WithErrorHandler(handler func(error)) MetricHandlerOption {
	return func(m *MetricHandler) {
		m.errorHandler = handler
	}
}

// NewMetricHandler creates a new MetricHandler.
func NewMetricHandler(m metric.Meter, opts ...MetricHandlerOption) (*MetricHandler, error) {
	if m == nil {
		return nil, ErrNilMeter
	}

	handler := &MetricHandler{
		meter:       m,
		recordFuncs: make(map[event.Metric]recordFunc),
		errorHandler: func(err error) {
			log.Printf("MetricHandler error: %v", err)
		},
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler, nil
}

func (m *MetricHandler) Event(ctx context.Context, e *event.Event) context.Context {
	if e == nil {
		m.handleError(ErrNilEvent)
		return ctx
	}

	if e.Kind != event.MetricKind {
		return ctx
	}

	// Get the otel instrument corresponding to the event's MetricDescriptor,
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

	rf, err := m.getRecordFunc(em)
	if err != nil {
		m.handleError(fmt.Errorf("unable to get record function for metric %v: %w", em.Name(), err))
		return ctx
	}

	if err := rf(ctx, lval, e.Labels); err != nil {
		m.handleError(fmt.Errorf("failed to record metric %v: %w", em.Name(), err))
	}

	return ctx
}

func (m *MetricHandler) getRecordFunc(em event.Metric) (recordFunc, error) {
	// First try read lock for existing function
	m.mu.RLock()
	if f, ok := m.recordFuncs[em]; ok {
		m.mu.RUnlock()
		return f, nil
	}
	m.mu.RUnlock()

	// Upgrade to write lock to create new function
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check pattern
	if f, ok := m.recordFuncs[em]; ok {
		return f, nil
	}

	f, err := m.newRecordFunc(em)
	if err != nil {
		return nil, err
	}

	m.recordFuncs[em] = f
	return f, nil
}

func (m *MetricHandler) newRecordFunc(em event.Metric) (recordFunc, error) {
	if em == nil {
		return nil, errors.New("metric cannot be nil")
	}

	opts := em.Options()
	name := em.Name()

	// Validate metric name
	if name == "" {
		return nil, errors.New("metric name cannot be empty")
	}

	// Add namespace prefix if provided
	if opts.Namespace != "" {
		name = opts.Namespace + "_" + name
	}

	switch metricType := em.(type) {
	case *event.Counter:
		otelOpts := []metric.Int64CounterOption{
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)), // cast OK: same strings
		}
		c, err := m.meter.Int64Counter(name, otelOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create counter %s: %w", name, err)
		}
		return func(ctx context.Context, l event.Label, attrs []event.Label) error {
			value := l.Int64()
			if value < 0 {
				return fmt.Errorf("counter value cannot be negative: %d", value)
			}
			c.Add(ctx, value, metric.WithAttributes(labelsToAttributes(attrs)...))
			return nil
		}, nil

	case *event.FloatGauge:
		otelOpts := []metric.Float64UpDownCounterOption{
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)), // cast OK: same strings
		}
		g, err := m.meter.Float64UpDownCounter(name, otelOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gauge %s: %w", name, err)
		}
		return func(ctx context.Context, l event.Label, attrs []event.Label) error {
			g.Add(ctx, l.Float64(), metric.WithAttributes(labelsToAttributes(attrs)...))
			return nil
		}, nil

	case *event.DurationDistribution:
		otelOpts := []metric.Int64HistogramOption{
			metric.WithDescription(opts.Description),
			metric.WithUnit(string(opts.Unit)), // cast OK: same strings
		}
		r, err := m.meter.Int64Histogram(name, otelOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create histogram %s: %w", name, err)
		}
		return func(ctx context.Context, l event.Label, attrs []event.Label) error {
			duration := l.Duration()
			if duration < 0 {
				return fmt.Errorf("duration cannot be negative: %v", duration)
			}
			r.Record(ctx, duration.Nanoseconds(), metric.WithAttributes(labelsToAttributes(attrs)...))
			return nil
		}, nil

	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedMetricType, metricType)
	}
}

func (m *MetricHandler) handleError(err error) {
	if m.errorHandler != nil {
		m.errorHandler(err)
	}
}

// Close cleans up resources used by the MetricHandler.
// It's safe to call multiple times.
func (m *MetricHandler) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear the map to prevent memory leaks
	m.recordFuncs = make(map[event.Metric]recordFunc)
	return nil
}

func labelsToAttributes(ls []event.Label) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	for _, l := range ls {
		if l.Name == string(event.MetricKey) || l.Name == string(event.MetricVal) {
			continue
		}
		// Skip empty labels
		if l.Name == "" || !l.HasValue() {
			continue
		}
		attr, err := labelToAttribute(l)
		if err == nil {
			attrs = append(attrs, attr)
		}
	}
	return attrs
}

func labelToAttribute(l event.Label) (attribute.KeyValue, error) {
	switch {
	case l.IsString():
		return attribute.String(l.Name, l.String()), nil
	case l.IsInt64():
		return attribute.Int64(l.Name, l.Int64()), nil
	case l.IsFloat64():
		return attribute.Float64(l.Name, l.Float64()), nil
	case l.IsBool():
		return attribute.Bool(l.Name, l.Bool()), nil
	default: // including uint64
		return attribute.KeyValue{}, fmt.Errorf("cannot convert label value of type %T to attribute.KeyValue", l.Interface())
	}
}
