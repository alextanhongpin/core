package batch

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var ErrKeyNotExist = errors.New("batch: key does not exist")

// Metrics contains runtime metrics for the batch loader.
type Metrics struct {
	CacheHits   int64 // Number of successful cache hits
	CacheMisses int64 // Number of cache misses
	BatchCalls  int64 // Number of batch function calls
	TotalKeys   int64 // Total keys requested
	ErrorCount  int64 // Number of errors encountered
}

type LoaderOptions[K comparable, V any] struct {
	Cache   cache[K, *Result[V]]
	BatchFn func([]K) (map[K]V, error)
	TTL     time.Duration

	// Advanced options
	MaxBatchSize int                                               // Maximum keys per batch call
	OnCacheHit   func(keys []K)                                    // Called on cache hits
	OnCacheMiss  func(keys []K)                                    // Called on cache misses
	OnBatchCall  func(keys []K, duration time.Duration, err error) // Called after batch function
	OnError      func(key K, err error)                            // Called on individual key errors
}

func (o *LoaderOptions[K, V]) Valid() error {
	o.TTL = cmp.Or(o.TTL, time.Hour)
	if o.TTL <= 0 {
		return errors.New("batch: TTL must be greater than 0")
	}
	if o.BatchFn == nil {
		return errors.New("batch: BatchFn is required")
	}

	if o.Cache == nil {
		o.Cache = NewCache[K, *Result[V]]()
	}

	// Set default max batch size
	if o.MaxBatchSize <= 0 {
		o.MaxBatchSize = 100
	}

	return nil
}

// LoaderMetricsCollector defines the interface for collecting loader metrics.
type LoaderMetricsCollector interface {
	IncCacheHits(n int64)
	IncCacheMisses(n int64)
	IncBatchCalls(n int64)
	IncTotalKeys(n int64)
	IncErrorCount(n int64)
	GetMetrics() Metrics
}

// AtomicLoaderMetricsCollector is the default atomic-based metrics implementation.
type AtomicLoaderMetricsCollector struct {
	cacheHits   int64
	cacheMisses int64
	batchCalls  int64
	totalKeys   int64
	errorCount  int64
}

func (m *AtomicLoaderMetricsCollector) IncCacheHits(n int64)   { atomic.AddInt64(&m.cacheHits, n) }
func (m *AtomicLoaderMetricsCollector) IncCacheMisses(n int64) { atomic.AddInt64(&m.cacheMisses, n) }
func (m *AtomicLoaderMetricsCollector) IncBatchCalls(n int64)  { atomic.AddInt64(&m.batchCalls, n) }
func (m *AtomicLoaderMetricsCollector) IncTotalKeys(n int64)   { atomic.AddInt64(&m.totalKeys, n) }
func (m *AtomicLoaderMetricsCollector) IncErrorCount(n int64)  { atomic.AddInt64(&m.errorCount, n) }
func (m *AtomicLoaderMetricsCollector) GetMetrics() Metrics {
	return Metrics{
		CacheHits:   atomic.LoadInt64(&m.cacheHits),
		CacheMisses: atomic.LoadInt64(&m.cacheMisses),
		BatchCalls:  atomic.LoadInt64(&m.batchCalls),
		TotalKeys:   atomic.LoadInt64(&m.totalKeys),
		ErrorCount:  atomic.LoadInt64(&m.errorCount),
	}
}

// PrometheusLoaderMetricsCollector implements LoaderMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusLoaderMetricsCollector struct {
	CacheHits   prometheus.Counter
	CacheMisses prometheus.Counter
	BatchCalls  prometheus.Counter
	TotalKeys   prometheus.Counter
	ErrorCount  prometheus.Counter
}

func (m *PrometheusLoaderMetricsCollector) IncCacheHits(n int64)   { m.CacheHits.Add(float64(n)) }
func (m *PrometheusLoaderMetricsCollector) IncCacheMisses(n int64) { m.CacheMisses.Add(float64(n)) }
func (m *PrometheusLoaderMetricsCollector) IncBatchCalls(n int64)  { m.BatchCalls.Add(float64(n)) }
func (m *PrometheusLoaderMetricsCollector) IncTotalKeys(n int64)   { m.TotalKeys.Add(float64(n)) }
func (m *PrometheusLoaderMetricsCollector) IncErrorCount(n int64)  { m.ErrorCount.Add(float64(n)) }
func (m *PrometheusLoaderMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}

type Loader[K comparable, V any] struct {
	opts    *LoaderOptions[K, V]
	metrics LoaderMetricsCollector
}

func NewLoader[K comparable, V any](opts *LoaderOptions[K, V], metrics ...LoaderMetricsCollector) *Loader[K, V] {
	if err := opts.Valid(); err != nil {
		panic(err)
	}
	var m LoaderMetricsCollector
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	} else {
		m = &AtomicLoaderMetricsCollector{}
	}
	return &Loader[K, V]{
		opts:    opts,
		metrics: m,
	}
}

// Metrics returns a copy of the current metrics.
func (l *Loader[K, V]) Metrics() Metrics {
	return l.metrics.GetMetrics()
}

func (l *Loader[K, V]) Load(ctx context.Context, k K) (v V, err error) {
	m, err := l.LoadManyResult(ctx, []K{k})
	if err != nil {
		return v, err
	}

	return m[k].Unwrap()
}

func (l *Loader[K, V]) LoadMany(ctx context.Context, ks []K) ([]V, error) {
	m, err := l.LoadManyResult(ctx, ks)
	if err != nil {
		return nil, err
	}
	res := make([]V, 0, len(ks))
	for _, k := range ks {
		r, ok := m[k]
		if !ok {
			continue
		}
		v, err := r.Unwrap()
		if errors.Is(err, ErrKeyNotExist) {
			continue
		}
		res = append(res, v)
	}

	return res, nil
}

func (l *Loader[K, V]) LoadManyResult(ctx context.Context, ks []K) (map[K]*Result[V], error) {
	l.metrics.IncTotalKeys(int64(len(ks)))

	m, err := l.opts.Cache.LoadMany(ctx, ks...)
	if err != nil {
		l.metrics.IncErrorCount(1)
		return nil, err
	}

	pks := make([]K, 0, len(ks))
	res := make(map[K]*Result[V])

	// Separate cache hits from misses
	cacheHitKeys := make([]K, 0, len(ks))
	for _, k := range ks {
		if v, ok := m[k]; ok {
			res[k] = v
			cacheHitKeys = append(cacheHitKeys, k)
			continue
		}
		pks = append(pks, k)
	}

	// Update metrics and call callbacks
	l.metrics.IncCacheHits(int64(len(cacheHitKeys)))
	l.metrics.IncCacheMisses(int64(len(pks)))

	if len(cacheHitKeys) > 0 && l.opts.OnCacheHit != nil {
		l.opts.OnCacheHit(cacheHitKeys)
	}
	if len(pks) > 0 && l.opts.OnCacheMiss != nil {
		l.opts.OnCacheMiss(pks)
	}

	// All keys found in cache, return.
	if len(res) == len(ks) {
		return res, nil
	}

	// Process pending keys in batches
	return l.processPendingKeys(ctx, pks, res)
}

func (l *Loader[K, V]) processPendingKeys(ctx context.Context, pks []K, res map[K]*Result[V]) (map[K]*Result[V], error) {
	// Process keys in batches
	for i := 0; i < len(pks); i += l.opts.MaxBatchSize {
		end := i + l.opts.MaxBatchSize
		if end > len(pks) {
			end = len(pks)
		}
		batch := pks[i:end]

		if err := l.processBatch(ctx, batch, res); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (l *Loader[K, V]) processBatch(ctx context.Context, batch []K, res map[K]*Result[V]) error {
	start := time.Now()

	// Fetch the pending keys.
	b, err := l.opts.BatchFn(batch)
	duration := time.Since(start)

	l.metrics.IncBatchCalls(1)

	// Call batch callback
	if l.opts.OnBatchCall != nil {
		l.opts.OnBatchCall(batch, duration, err)
	}

	if err != nil {
		l.metrics.IncErrorCount(1)
		return err
	}

	// Stores the new keys in the cache.
	n := make(map[K]*Result[V])
	for _, k := range batch {
		v, ok := b[k]
		var result *Result[V]
		if ok {
			result = newResult(v, nil)
		} else {
			keyErr := newKeyError(fmt.Sprint(k), ErrKeyNotExist)
			result = newResult(v, keyErr)

			// Call error callback for individual key
			if l.opts.OnError != nil {
				l.opts.OnError(k, keyErr)
			}
		}
		n[k] = result
		res[k] = result
	}

	if err := l.opts.Cache.StoreMany(ctx, n, l.opts.TTL); err != nil {
		l.metrics.IncErrorCount(1)
		return err
	}

	return nil
}

type Result[T any] struct {
	data T
	err  error
}

func (r *Result[T]) Unwrap() (T, error) {
	return r.data, r.err
}

func newResult[T any](data T, err error) *Result[T] {
	return &Result[T]{
		data: data,
		err:  err,
	}
}

type KeyError struct {
	Key string
	Err error
}

func newKeyError(key string, err error) *KeyError {
	return &KeyError{
		Key: key,
		Err: err,
	}
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("%s: %q", e.Err, e.Key)
}

func (e *KeyError) Is(err error) bool {
	return errors.Is(e.Err, err)
}
