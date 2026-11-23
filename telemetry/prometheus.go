package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/event"
)

var (
	ErrNilRegisterer                   = errors.New("prometheus registerer cannot be nil")
	ErrUnsupportedPrometheusMetricType = errors.New("unsupported metric type for PrometheusHandler")
	ErrUnsupportedCollectorType        = errors.New("unsupported collector type for PrometheusHandler")
	ErrMetricConflict                  = errors.New("metric name already used by a different metric")
)

// PrometheusHandler is an event.Handler for Prometheus metrics.
// Its Event method handles Metric events and ignores all others.
//
// The label names of a metric are fixed by its first event; later events
// with a different label set are reported to the error handler and dropped.
type PrometheusHandler struct {
	client prometheus.Registerer
	mu     sync.RWMutex
	// collectors is keyed by event.Metric name.
	collectors map[string]*promCollector
	// errorHandler allows custom error handling instead of panicking
	errorHandler func(error)
}

type promCollector struct {
	c         prometheus.Collector
	kind      string
	namespace string
	// owned reports whether this handler registered c, and so may unregister it.
	owned bool
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
		collectors: make(map[string]*promCollector),
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
	labels := labelsToPrometheus(e.Labels)

	c, err := m.collector(em, &opts, slices.Sorted(maps.Keys(labels)))
	if err != nil {
		m.handleError(fmt.Errorf("failed to ensure collector for %s: %w", name, err))
		return ctx
	}

	if err := record(c, em, &opts, lval, labels); err != nil {
		m.handleError(fmt.Errorf("failed to record metric %s: %w", name, err))
	}

	return ctx
}

// collector returns the collector for em, creating and registering it on
// first use with the given label names.
func (m *PrometheusHandler) collector(em event.Metric, opts *event.MetricOptions, keys []string) (prometheus.Collector, error) {
	name := em.Name()
	kind := fmt.Sprintf("%T", em)
	namespace := opts.Namespace

	m.mu.RLock()
	pc, ok := m.collectors[name]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		defer m.mu.Unlock()

		pc, ok = m.collectors[name]
		if !ok {
			var err error
			pc, err = m.register(em, name, namespace, opts, keys)
			if err != nil {
				return nil, err
			}
			pc.kind, pc.namespace = kind, namespace
			m.collectors[name] = pc
		}
	}

	if pc.kind != kind || pc.namespace != namespace {
		return nil, fmt.Errorf("%w: %s is a %s in namespace %q", ErrMetricConflict, name, pc.kind, pc.namespace)
	}

	return pc.c, nil
}

func (m *PrometheusHandler) register(em event.Metric, name, namespace string, opts *event.MetricOptions, keys []string) (*promCollector, error) {
	if opts.Unit == event.UnitBytes {
		name += "_bytes"
	}

	var c prometheus.Collector
	switch em.(type) {
	case *event.Counter:
		c = prometheus.NewCounterVec(prometheus.CounterOpts{
			Help:      opts.Description,
			Name:      name,
			Namespace: namespace,
		}, keys)
	case *event.FloatGauge:
		c = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Help:      opts.Description,
			Name:      name,
			Namespace: namespace,
		}, keys)
	case *event.DurationDistribution:
		if opts.Unit == event.UnitMilliseconds {
			name += "_milliseconds"
		} else {
			name += "_seconds"
		}
		c = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Help:      opts.Description,
			Name:      name,
			Namespace: namespace,
		}, keys)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPrometheusMetricType, em.Name())
	}

	if err := m.client.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			// Share the existing collector, but leave its registration to
			// whoever registered it.
			return &promCollector{c: are.ExistingCollector}, nil
		}
		return nil, fmt.Errorf("failed to register collector %s: %w", name, err)
	}

	return &promCollector{c: c, owned: true}, nil
}

func record(c prometheus.Collector, em event.Metric, opts *event.MetricOptions, lval event.Label, labels prometheus.Labels) error {
	switch col := c.(type) {
	case *prometheus.CounterVec:
		if _, ok := em.(*event.Counter); !ok {
			break
		}
		value := lval.Int64()
		if value < 0 {
			return fmt.Errorf("counter value cannot be negative: %d", value)
		}
		m, err := col.GetMetricWith(labels)
		if err != nil {
			return err
		}
		m.Add(float64(value))
		return nil
	case *prometheus.GaugeVec:
		if _, ok := em.(*event.FloatGauge); !ok {
			break
		}
		m, err := col.GetMetricWith(labels)
		if err != nil {
			return err
		}
		m.Set(lval.Float64())
		return nil
	case *prometheus.HistogramVec:
		if _, ok := em.(*event.DurationDistribution); !ok {
			break
		}
		duration := lval.Duration()
		if duration < 0 {
			return fmt.Errorf("duration cannot be negative: %v", duration)
		}
		scale := float64(time.Second)
		if opts.Unit == event.UnitMilliseconds {
			scale = float64(time.Millisecond)
		}
		m, err := col.GetMetricWith(labels)
		if err != nil {
			return err
		}
		m.Observe(float64(duration) / scale)
		return nil
	}

	return fmt.Errorf("%w: %T for %T", ErrUnsupportedCollectorType, c, em)
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

	pc, ok := m.collectors[name]
	if !ok {
		return nil, false
	}
	return pc.c, true
}

// Close unregisters the collectors this handler registered. Collectors that
// were already registered by someone else are left in place.
// It's safe to call multiple times.
func (m *PrometheusHandler) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, pc := range m.collectors {
		if pc.owned && !m.client.Unregister(pc.c) {
			m.handleError(fmt.Errorf("failed to unregister collector %s", name))
		}
	}

	m.collectors = make(map[string]*promCollector)
	return nil
}

func labelsToPrometheus(ls []event.Label) prometheus.Labels {
	labels := make(prometheus.Labels, len(ls))
	for _, l := range ls {
		if l.Name == string(event.MetricKey) || l.Name == string(event.MetricVal) {
			continue
		}
		// Skip empty labels
		if l.Name == "" || !l.HasValue() {
			continue
		}
		labels[l.Name] = l.String()
	}
	return labels
}
