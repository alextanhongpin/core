package lock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ErrTimeout is returned when a lock operation times out.
	ErrTimeout = errors.New("lock: timeout")

	// ErrCanceled is returned when a context is canceled.
	ErrCanceled = errors.New("lock: canceled")

	// ErrInvalidKey is returned when an empty or invalid key is provided.
	ErrInvalidKey = errors.New("lock: invalid key")
)

// LockType represents the type of lock to create.
type LockType int

const (
	// Mutex creates a standard mutex lock.
	Mutex LockType = iota
	// RWMutex creates a read-write mutex lock.
	RWMutex
)

// Metrics contains runtime metrics for the lock manager.
type Metrics struct {
	ActiveLocks      int64         // Number of active locks
	TotalLocks       int64         // Total locks created
	LockAcquisitions int64         // Total lock acquisitions
	LockContentions  int64         // Number of lock contentions
	AverageWaitTime  time.Duration // Average wait time for locks
	MaxWaitTime      time.Duration // Maximum wait time observed
}

// Options configures the lock manager behavior.
type Options struct {
	// DefaultTimeout is the default timeout for lock operations.
	DefaultTimeout time.Duration

	// CleanupInterval is how often to run cleanup for unused locks.
	CleanupInterval time.Duration

	// LockType specifies the type of locks to create.
	LockType LockType

	// MaxLocks is the maximum number of locks to keep in memory.
	MaxLocks int

	// OnLockAcquired is called when a lock is acquired.
	OnLockAcquired func(key string, waitTime time.Duration)

	// OnLockReleased is called when a lock is released.
	OnLockReleased func(key string, holdTime time.Duration)

	// OnLockContention is called when lock contention is detected.
	OnLockContention func(key string, waitingGoroutines int)
}

// lockEntry represents a lock with reference counting and metrics.
type lockEntry struct {
	locker    sync.Locker
	rwLocker  *sync.RWMutex // Only set if LockType is RWMutex
	refCount  int64
	createdAt time.Time
	lastUsed  int64 // Unix timestamp
	waiters   int64 // Number of waiting goroutines
}

// LockMetricsCollector defines the interface for collecting lock metrics.
type LockMetricsCollector interface {
	IncActiveLocks()
	DecActiveLocks()
	IncTotalLocks()
	IncLockAcquisitions()
	IncLockContentions()
	AddWaitTime(d time.Duration)
	SetMaxWaitTime(d time.Duration)
	GetMetrics() Metrics
}

// AtomicLockMetricsCollector is the default atomic-based metrics implementation.
type AtomicLockMetricsCollector struct {
	activeLocks      int64
	totalLocks       int64
	lockAcquisitions int64
	lockContentions  int64
	totalWaitTime    int64 // nanoseconds
	maxWaitTime      int64 // nanoseconds
}

func (m *AtomicLockMetricsCollector) IncActiveLocks()      { atomic.AddInt64(&m.activeLocks, 1) }
func (m *AtomicLockMetricsCollector) DecActiveLocks()      { atomic.AddInt64(&m.activeLocks, -1) }
func (m *AtomicLockMetricsCollector) IncTotalLocks()       { atomic.AddInt64(&m.totalLocks, 1) }
func (m *AtomicLockMetricsCollector) IncLockAcquisitions() { atomic.AddInt64(&m.lockAcquisitions, 1) }
func (m *AtomicLockMetricsCollector) IncLockContentions()  { atomic.AddInt64(&m.lockContentions, 1) }
func (m *AtomicLockMetricsCollector) AddWaitTime(d time.Duration) {
	atomic.AddInt64(&m.totalWaitTime, int64(d))
}
func (m *AtomicLockMetricsCollector) SetMaxWaitTime(d time.Duration) {
	for {
		current := atomic.LoadInt64(&m.maxWaitTime)
		if int64(d) <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.maxWaitTime, current, int64(d)) {
			break
		}
	}
}
func (m *AtomicLockMetricsCollector) GetMetrics() Metrics {
	acquisitions := atomic.LoadInt64(&m.lockAcquisitions)
	var avgWaitTime time.Duration
	if acquisitions > 0 {
		avgWaitTime = time.Duration(atomic.LoadInt64(&m.totalWaitTime) / acquisitions)
	}
	return Metrics{
		ActiveLocks:      atomic.LoadInt64(&m.activeLocks),
		TotalLocks:       atomic.LoadInt64(&m.totalLocks),
		LockAcquisitions: acquisitions,
		LockContentions:  atomic.LoadInt64(&m.lockContentions),
		AverageWaitTime:  avgWaitTime,
		MaxWaitTime:      time.Duration(atomic.LoadInt64(&m.maxWaitTime)),
	}
}

// PrometheusLockMetricsCollector implements LockMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusLockMetricsCollector struct {
	ActiveLocks      prometheus.Gauge
	TotalLocks       prometheus.Counter
	LockAcquisitions prometheus.Counter
	LockContentions  prometheus.Counter
	TotalWaitTime    prometheus.Counter
	MaxWaitTime      prometheus.Gauge
}

func (m *PrometheusLockMetricsCollector) IncActiveLocks()      { m.ActiveLocks.Inc() }
func (m *PrometheusLockMetricsCollector) DecActiveLocks()      { m.ActiveLocks.Dec() }
func (m *PrometheusLockMetricsCollector) IncTotalLocks()       { m.TotalLocks.Inc() }
func (m *PrometheusLockMetricsCollector) IncLockAcquisitions() { m.LockAcquisitions.Inc() }
func (m *PrometheusLockMetricsCollector) IncLockContentions()  { m.LockContentions.Inc() }
func (m *PrometheusLockMetricsCollector) AddWaitTime(d time.Duration) {
	m.TotalWaitTime.Add(float64(d.Nanoseconds()))
}
func (m *PrometheusLockMetricsCollector) SetMaxWaitTime(d time.Duration) {
	m.MaxWaitTime.Set(float64(d.Nanoseconds()))
}
func (m *PrometheusLockMetricsCollector) GetMetrics() Metrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return Metrics{}
}

// Lock provides named locks with automatic cleanup and metrics.
type Lock struct {
	mu   sync.RWMutex
	data map[string]*lockEntry
	opts Options

	metrics LockMetricsCollector

	// Cleanup management
	stopCleanup chan struct{}
	cleanupOnce sync.Once
}

// New creates a new lock manager with default options.
func New() *Lock {
	return NewWithOptions(Options{
		DefaultTimeout:  30 * time.Second,
		CleanupInterval: 5 * time.Minute,
		LockType:        Mutex,
		MaxLocks:        10000,
	})
}

// NewWithOptions creates a new lock manager with custom options.
func NewWithOptions(opts Options, metrics ...LockMetricsCollector) *Lock {
	if opts.DefaultTimeout <= 0 {
		opts.DefaultTimeout = 30 * time.Second
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = 5 * time.Minute
	}
	if opts.MaxLocks <= 0 {
		opts.MaxLocks = 10000
	}

	var m LockMetricsCollector
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	} else {
		m = &AtomicLockMetricsCollector{}
	}

	l := &Lock{
		data:        make(map[string]*lockEntry),
		opts:        opts,
		stopCleanup: make(chan struct{}),
		metrics:     m,
	}

	// Start cleanup goroutine
	go l.cleanupLoop()

	return l
}

// Get returns a locker for the given key.
func (l *Lock) Get(key string) sync.Locker {
	if key == "" {
		panic(ErrInvalidKey)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.data[key]
	if !exists {
		entry = l.createLockEntry()
		l.data[key] = entry
		l.metrics.IncTotalLocks()
		l.metrics.IncActiveLocks()
	}

	atomic.AddInt64(&entry.refCount, 1)
	atomic.StoreInt64(&entry.lastUsed, time.Now().Unix())

	return &lockWrapper{
		entry: entry,
		key:   key,
		lock:  l,
	}
}

// GetRW returns a read-write locker for the given key (only if LockType is RWMutex).
func (l *Lock) GetRW(key string) *sync.RWMutex {
	if key == "" {
		panic(ErrInvalidKey)
	}
	if l.opts.LockType != RWMutex {
		panic("lock: GetRW can only be used with RWMutex lock type")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.data[key]
	if !exists {
		entry = l.createLockEntry()
		l.data[key] = entry
		l.metrics.IncTotalLocks()
		l.metrics.IncActiveLocks()
	}

	atomic.AddInt64(&entry.refCount, 1)
	atomic.StoreInt64(&entry.lastUsed, time.Now().Unix())

	return entry.rwLocker
}

// LockWithTimeout attempts to acquire a lock with a timeout.
func (l *Lock) LockWithTimeout(key string, timeout time.Duration) (func(), error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	locker := l.Get(key)

	done := make(chan struct{})
	var acquired int32

	go func() {
		locker.Lock()
		atomic.StoreInt32(&acquired, 1)
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return func() { locker.Unlock() }, nil
	case <-timer.C:
		// Note: If the lock is acquired after timeout, it will be unlocked automatically
		go func() {
			<-done
			if atomic.LoadInt32(&acquired) == 1 {
				locker.Unlock()
			}
		}()
		return nil, ErrTimeout
	}
}

// LockWithContext attempts to acquire a lock that can be canceled.
func (l *Lock) LockWithContext(ctx context.Context, key string) (func(), error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	locker := l.Get(key)

	done := make(chan struct{})
	var acquired int32

	go func() {
		locker.Lock()
		atomic.StoreInt32(&acquired, 1)
		close(done)
	}()

	select {
	case <-done:
		return func() { locker.Unlock() }, nil
	case <-ctx.Done():
		// Note: If the lock is acquired after cancellation, it will be unlocked automatically
		go func() {
			<-done
			if atomic.LoadInt32(&acquired) == 1 {
				locker.Unlock()
			}
		}()
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, ErrCanceled
	}
}

// Metrics returns a copy of the current metrics.
func (l *Lock) Metrics() Metrics {
	return l.metrics.GetMetrics()
}

// Stop gracefully stops the lock manager and cleanup goroutine.
func (l *Lock) Stop() {
	l.cleanupOnce.Do(func() {
		close(l.stopCleanup)
	})
}

// Size returns the current number of locks in memory.
func (l *Lock) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.data)
}

// createLockEntry creates a new lock entry based on the configured lock type.
func (l *Lock) createLockEntry() *lockEntry {
	entry := &lockEntry{
		createdAt: time.Now(),
		lastUsed:  time.Now().Unix(),
	}

	switch l.opts.LockType {
	case Mutex:
		entry.locker = &sync.Mutex{}
	case RWMutex:
		rwMutex := &sync.RWMutex{}
		entry.locker = rwMutex
		entry.rwLocker = rwMutex
	}

	return entry
}

// cleanupLoop periodically removes unused locks.
func (l *Lock) cleanupLoop() {
	ticker := time.NewTicker(l.opts.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCleanup:
			return
		}
	}
}

// cleanup removes locks that haven't been used recently and have no references.
func (l *Lock) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Unix()
	cleanupThreshold := int64(l.opts.CleanupInterval.Seconds())

	for key, entry := range l.data {
		// Remove if no references and not used recently
		if atomic.LoadInt64(&entry.refCount) == 0 &&
			now-atomic.LoadInt64(&entry.lastUsed) > cleanupThreshold {
			delete(l.data, key)
			l.metrics.DecActiveLocks()
		}
	}

	// If we still have too many locks, remove the oldest ones
	if len(l.data) > l.opts.MaxLocks {
		// Simple cleanup: remove excess locks (in production, you might want LRU)
		excess := len(l.data) - l.opts.MaxLocks
		count := 0
		for key := range l.data {
			if count >= excess {
				break
			}
			delete(l.data, key)
			l.metrics.DecActiveLocks()
			count++
		}
	}
}

// lockWrapper wraps a lock entry to provide metrics and callbacks.
type lockWrapper struct {
	entry     *lockEntry
	key       string
	lock      *Lock
	startTime time.Time
}

func (lw *lockWrapper) Lock() {
	lw.startTime = time.Now()

	// Track contention
	waiters := atomic.AddInt64(&lw.entry.waiters, 1)
	if waiters > 1 {
		lw.lock.metrics.IncLockContentions()
		if lw.lock.opts.OnLockContention != nil {
			lw.lock.opts.OnLockContention(lw.key, int(waiters))
		}
	}

	lw.entry.locker.Lock()

	atomic.AddInt64(&lw.entry.waiters, -1)
	lw.lock.metrics.IncLockAcquisitions()

	// Track wait time
	waitTime := time.Since(lw.startTime)
	lw.lock.metrics.AddWaitTime(waitTime)

	// Update max wait time
	lw.lock.metrics.SetMaxWaitTime(waitTime)

	if lw.lock.opts.OnLockAcquired != nil {
		lw.lock.opts.OnLockAcquired(lw.key, waitTime)
	}
}

func (lw *lockWrapper) Unlock() {
	lw.entry.locker.Unlock()
	atomic.AddInt64(&lw.entry.refCount, -1)

	if lw.lock.opts.OnLockReleased != nil {
		holdTime := time.Since(lw.startTime)
		lw.lock.opts.OnLockReleased(lw.key, holdTime)
	}
}
