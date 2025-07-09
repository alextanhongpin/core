package dataloader

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
	"github.com/alextanhongpin/core/sync/promise"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ErrNoResult is returned if the key does not have a value.
	// This might not necessarily be an error, for example, if the row is not
	// found in the database when performing `SELECT ... IN (k1, k2, ... kn)`.
	ErrNoResult = errors.New("dataloader: no result")

	// ErrTerminated is returned when the dataloader is terminated.
	ErrTerminated = errors.New("dataloader: terminated")

	// ErrTimeout is returned when a load operation times out.
	ErrTimeout = errors.New("dataloader: operation timeout")

	// ErrInvalidKey is returned when an invalid key is provided.
	ErrInvalidKey = errors.New("dataloader: invalid key")
)

// Metrics contains runtime metrics for the dataloader.
type Metrics struct {
	// Total number of Load/LoadMany requests
	TotalRequests int64

	// Number of keys requested
	KeysRequested int64

	// Number of keys loaded from cache
	CacheHits int64

	// Number of keys loaded from batch function
	CacheMisses int64

	// Number of batch operations performed
	BatchCalls int64

	// Number of errors encountered
	ErrorCount int64

	// Number of keys with no result
	NoResultCount int64

	// Current cache size
	CacheSize int64

	// Queue length
	QueueLength int64
}

type Options[K comparable, V any] struct {
	// BatchFn is the function that loads multiple keys at once.
	// It should return a map where keys map to their values.
	// Keys not present in the map will be treated as ErrNoResult.
	BatchFn func(ctx context.Context, keys []K) (map[K]V, error)

	// BatchMaxKeys is the maximum number of keys to batch together.
	// Default is 1000.
	BatchMaxKeys int

	// BatchTimeout is the maximum time to wait before executing a batch.
	// Default is 16ms.
	BatchTimeout time.Duration

	// BatchQueueSize is the size of the batch queue.
	// Default is 0 (unbounded).
	BatchQueueSize int

	// Cache is the cache implementation to use.
	// Default is a simple in-memory cache.
	Cache cache[K, V]

	// LoadTimeout is the maximum time to wait for a single load operation.
	// Default is 30 seconds.
	LoadTimeout time.Duration

	// MaxCacheSize is the maximum number of items to keep in cache.
	// Default is 0 (unbounded).
	MaxCacheSize int

	// OnBatchStart is called before a batch operation starts.
	OnBatchStart func(keys []K)

	// OnBatchComplete is called after a batch operation completes.
	OnBatchComplete func(keys []K, duration time.Duration, err error)

	// OnCacheHit is called when a key is found in cache.
	OnCacheHit func(key K)

	// OnCacheMiss is called when a key is not found in cache.
	OnCacheMiss func(key K)

	// OnError is called when an error occurs.
	OnError func(key K, err error)
}

func (o *Options[K, V]) Valid() error {
	if o.BatchFn == nil {
		return errors.New("dataloader: BatchFn is required")
	}

	o.BatchMaxKeys = cmp.Or(o.BatchMaxKeys, 1_000)
	if o.BatchMaxKeys < 1 {
		return errors.New("dataloader: BatchMaxKeys must be greater than zero")
	}

	o.BatchTimeout = cmp.Or(o.BatchTimeout, 16*time.Millisecond)
	if o.BatchTimeout < 1 {
		return errors.New("dataloader: BatchTimeout must be greater than zero")
	}

	o.LoadTimeout = cmp.Or(o.LoadTimeout, 30*time.Second)
	if o.LoadTimeout < 1 {
		return errors.New("dataloader: LoadTimeout must be greater than zero")
	}

	if o.Cache == nil {
		o.Cache = NewCache[K, V]()
	}

	return nil
}

// DataLoaderMetricsCollector defines the interface for collecting dataloader metrics.
type DataLoaderMetricsCollector interface {
	IncTotalRequests()
	IncKeysRequested(n int64)
	IncCacheHits(n int64)
	IncCacheMisses(n int64)
	IncBatchCalls()
	IncErrorCount()
	IncNoResultCount()
	SetCacheSize(n int64)
	SetQueueLength(n int64)
	GetMetrics() Metrics
}

// AtomicDataLoaderMetricsCollector is the default atomic-based metrics implementation.
type AtomicDataLoaderMetricsCollector struct {
	totalRequests int64
	keysRequested int64
	cacheHits     int64
	cacheMisses   int64
	batchCalls    int64
	errorCount    int64
	noResultCount int64
	cacheSize     int64
	queueLength   int64
}

func (m *AtomicDataLoaderMetricsCollector) IncTotalRequests() { atomic.AddInt64(&m.totalRequests, 1) }
func (m *AtomicDataLoaderMetricsCollector) IncKeysRequested(n int64) {
	atomic.AddInt64(&m.keysRequested, n)
}
func (m *AtomicDataLoaderMetricsCollector) IncCacheHits(n int64) { atomic.AddInt64(&m.cacheHits, n) }
func (m *AtomicDataLoaderMetricsCollector) IncCacheMisses(n int64) {
	atomic.AddInt64(&m.cacheMisses, n)
}
func (m *AtomicDataLoaderMetricsCollector) IncBatchCalls()       { atomic.AddInt64(&m.batchCalls, 1) }
func (m *AtomicDataLoaderMetricsCollector) IncErrorCount()       { atomic.AddInt64(&m.errorCount, 1) }
func (m *AtomicDataLoaderMetricsCollector) IncNoResultCount()    { atomic.AddInt64(&m.noResultCount, 1) }
func (m *AtomicDataLoaderMetricsCollector) SetCacheSize(n int64) { atomic.StoreInt64(&m.cacheSize, n) }
func (m *AtomicDataLoaderMetricsCollector) SetQueueLength(n int64) {
	atomic.StoreInt64(&m.queueLength, n)
}
func (m *AtomicDataLoaderMetricsCollector) GetMetrics() Metrics {
	return Metrics{
		TotalRequests: atomic.LoadInt64(&m.totalRequests),
		KeysRequested: atomic.LoadInt64(&m.keysRequested),
		CacheHits:     atomic.LoadInt64(&m.cacheHits),
		CacheMisses:   atomic.LoadInt64(&m.cacheMisses),
		BatchCalls:    atomic.LoadInt64(&m.batchCalls),
		ErrorCount:    atomic.LoadInt64(&m.errorCount),
		NoResultCount: atomic.LoadInt64(&m.noResultCount),
		CacheSize:     atomic.LoadInt64(&m.cacheSize),
		QueueLength:   atomic.LoadInt64(&m.queueLength),
	}
}

// PrometheusDataLoaderMetricsCollector implements DataLoaderMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusDataLoaderMetricsCollector struct {
	TotalRequests prometheus.Counter
	KeysRequested prometheus.Counter
	CacheHits     prometheus.Counter
	CacheMisses   prometheus.Counter
	BatchCalls    prometheus.Counter
	ErrorCount    prometheus.Counter
	NoResultCount prometheus.Counter
	CacheSize     prometheus.Gauge
	QueueLength   prometheus.Gauge
}

func (m *PrometheusDataLoaderMetricsCollector) IncTotalRequests() { m.TotalRequests.Inc() }
func (m *PrometheusDataLoaderMetricsCollector) IncKeysRequested(n int64) {
	m.KeysRequested.Add(float64(n))
}
func (m *PrometheusDataLoaderMetricsCollector) IncCacheHits(n int64)   { m.CacheHits.Add(float64(n)) }
func (m *PrometheusDataLoaderMetricsCollector) IncCacheMisses(n int64) { m.CacheMisses.Add(float64(n)) }
func (m *PrometheusDataLoaderMetricsCollector) IncBatchCalls()         { m.BatchCalls.Inc() }
func (m *PrometheusDataLoaderMetricsCollector) IncErrorCount()         { m.ErrorCount.Inc() }
func (m *PrometheusDataLoaderMetricsCollector) IncNoResultCount()      { m.NoResultCount.Inc() }
func (m *PrometheusDataLoaderMetricsCollector) SetCacheSize(n int64)   { m.CacheSize.Set(float64(n)) }
func (m *PrometheusDataLoaderMetricsCollector) SetQueueLength(n int64) { m.QueueLength.Set(float64(n)) }
func (m *PrometheusDataLoaderMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}

type DataLoader[K comparable, V any] struct {
	// Lifecycle control.
	begin sync.Once
	end   sync.Once

	// Concurrency primitives.
	wg sync.WaitGroup

	// State.
	group  *promise.Map[V]
	ch     chan K
	ctx    context.Context
	cancel func(error)

	// Options.
	opts *Options[K, V]

	// Metrics (using atomic operations for thread safety)
	metrics     DataLoaderMetricsCollector
	queueLength int64 // local field for queue length
}

// New returns a new DataLoader. The context is passed in to control the lifecycle.
func New[K comparable, V any](ctx context.Context, opts *Options[K, V], metrics ...DataLoaderMetricsCollector) *DataLoader[K, V] {
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancelCause(ctx)
	var m DataLoaderMetricsCollector
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	} else {
		m = &AtomicDataLoaderMetricsCollector{}
	}
	return &DataLoader[K, V]{
		group:   promise.NewMap[V](),
		ch:      make(chan K),
		opts:    opts,
		ctx:     ctx,
		cancel:  cancel,
		metrics: m,
	}
}

// Set sets the key-value after expiring existing references.
// There is no use passing context as the first argument as it does not control
// the lifecycle.
func (d *DataLoader[K, V]) Set(k K, v V) {
	d.opts.Cache.Set(k, v, nil)
}

func (d *DataLoader[K, V]) Load(k K) (V, error) {
	d.metrics.IncTotalRequests()
	d.metrics.IncKeysRequested(1)
	return d.load(k).Await()
}

func (d *DataLoader[K, V]) LoadMany(ks []K) ([]promise.Result[V], error) {
	d.metrics.IncTotalRequests()
	d.metrics.IncKeysRequested(int64(len(ks)))
	res := make(promise.Promises[V], len(ks))
	for i, k := range ks {
		res[i] = d.load(k)
	}
	return res.AllSettled(), nil
}

func (d *DataLoader[K, V]) Stop() {
	d.end.Do(func() {
		// Make sure the dataloader is started before stopping it.
		d.begin.Do(func() {})
		d.cancel(ErrTerminated)

		d.wg.Wait()
	})
}

func (d *DataLoader[K, V]) start() {
	d.begin.Do(func() {
		d.wg.Add(1)

		go func() {
			defer d.wg.Done()

			d.loop()
		}()
	})
}

func (d *DataLoader[K, V]) load(k K) *promise.Promise[V] {
	ctx := d.ctx
	d.start()

	v, err := d.opts.Cache.Get(k)
	if err == nil {
		d.metrics.IncCacheHits(1)
		if d.opts.OnCacheHit != nil {
			d.opts.OnCacheHit(k)
		}
		return promise.Resolve(v)
	}

	if !errors.Is(err, ErrNotExist) {
		d.metrics.IncErrorCount()
		if d.opts.OnError != nil {
			d.opts.OnError(k, err)
		}
		return promise.Reject[V](err)
	}

	d.metrics.IncCacheMisses(1)
	if d.opts.OnCacheMiss != nil {
		d.opts.OnCacheMiss(k)
	}

	key := fmt.Sprint(k)
	p, loaded := d.group.LoadOrStore(key)
	if loaded {
		return p
	}

	select {
	case <-ctx.Done():
		err := newKeyError(k, context.Cause(ctx))
		d.opts.Cache.Set(k, v, err)
		p.Reject(err)
		d.group.Delete(fmt.Sprint(k))
	case d.ch <- k:
		d.queueLength++
		d.metrics.SetQueueLength(d.queueLength)
	}

	return p
}

func (d *DataLoader[K, V]) loop() {
	p1 := pipeline.Context(d.ctx, d.ch)
	p2 := pipeline.Batch(d.opts.BatchMaxKeys, d.opts.BatchTimeout, p1)
	p3 := pipeline.Queue(d.opts.BatchQueueSize, p2)

	for vs := range p3 {
		d.batch(d.ctx, vs)
	}
}

func (d *DataLoader[K, V]) batch(ctx context.Context, keys []K) {
	d.metrics.IncBatchCalls()
	if d.opts.OnBatchStart != nil {
		d.opts.OnBatchStart(keys)
	}
	start := time.Now()

	kv, err := d.opts.BatchFn(ctx, keys)

	if d.opts.OnBatchComplete != nil {
		duration := time.Since(start)
		d.opts.OnBatchComplete(keys, duration, err)
	}

	for _, k := range keys {
		fn := func() (V, error) {
			if err != nil {
				var v V
				return v, newKeyError(k, err)
			}

			v, ok := kv[k]
			if ok {
				return v, nil
			}

			d.metrics.IncNoResultCount()
			return v, newKeyError(k, ErrNoResult)
		}
		v, err := fn()
		d.opts.Cache.Set(k, v, err)

		key := fmt.Sprint(k)
		p, ok := d.group.Load(key)
		if ok {
			if err != nil {
				p.Reject(err)
			} else {
				p.Resolve(v)
			}
			d.group.Delete(key)
		}
	}
	// After processing a batch, decrement queue length
	d.queueLength -= int64(len(keys))
	d.metrics.SetQueueLength(d.queueLength)
}

func newKeyError(k any, err error) *KeyError {
	return &KeyError{
		Key: fmt.Sprint(k),
		err: err,
	}
}

type KeyError struct {
	Key string
	err error
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("dataloader.KeyError(%s): %s", e.Key, e.err)
}

func (e *KeyError) Is(err error) bool {
	return errors.Is(e.err, err)
}

func (e *KeyError) Unwrap() error {
	return e.err
}

// Metrics returns a copy of the current metrics.
func (d *DataLoader[K, V]) Metrics() Metrics {
	return d.metrics.GetMetrics()
}

// LoadWithTimeout loads a single key with a timeout.
func (d *DataLoader[K, V]) LoadWithTimeout(ctx context.Context, k K) (V, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, d.opts.LoadTimeout)
	defer cancel()

	promise := d.load(k)

	// Use a goroutine to handle the promise result
	type result struct {
		value V
		err   error
	}
	resultCh := make(chan result, 1)
	go func() {
		value, err := promise.Await()
		resultCh <- result{value: value, err: err}
	}()

	select {
	case <-timeoutCtx.Done():
		var v V
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			return v, ErrTimeout
		}
		return v, timeoutCtx.Err()
	case r := <-resultCh:
		return r.value, r.err
	}
}

// ClearCache removes all entries from the cache.
func (d *DataLoader[K, V]) ClearCache() {
	if clearer, ok := d.opts.Cache.(interface{ Clear() }); ok {
		clearer.Clear()
		d.metrics.SetCacheSize(0)
	}
}
