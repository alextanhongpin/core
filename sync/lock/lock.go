package lock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
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

// Lock provides named locks with automatic cleanup and metrics.
type Lock struct {
	mu   sync.RWMutex
	data map[string]*lockEntry
	opts Options

	// Metrics (using atomic operations for thread safety)
	metrics struct {
		activeLocks      int64
		totalLocks       int64
		lockAcquisitions int64
		lockContentions  int64
		totalWaitTime    int64 // In nanoseconds
		maxWaitTime      int64 // In nanoseconds
	}

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
func NewWithOptions(opts Options) *Lock {
	if opts.DefaultTimeout <= 0 {
		opts.DefaultTimeout = 30 * time.Second
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = 5 * time.Minute
	}
	if opts.MaxLocks <= 0 {
		opts.MaxLocks = 10000
	}

	l := &Lock{
		data:        make(map[string]*lockEntry),
		opts:        opts,
		stopCleanup: make(chan struct{}),
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
		atomic.AddInt64(&l.metrics.totalLocks, 1)
		atomic.AddInt64(&l.metrics.activeLocks, 1)
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
		atomic.AddInt64(&l.metrics.totalLocks, 1)
		atomic.AddInt64(&l.metrics.activeLocks, 1)
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
	totalWaitTime := atomic.LoadInt64(&l.metrics.totalWaitTime)
	acquisitions := atomic.LoadInt64(&l.metrics.lockAcquisitions)

	var avgWaitTime time.Duration
	if acquisitions > 0 {
		avgWaitTime = time.Duration(totalWaitTime / acquisitions)
	}

	return Metrics{
		ActiveLocks:      atomic.LoadInt64(&l.metrics.activeLocks),
		TotalLocks:       atomic.LoadInt64(&l.metrics.totalLocks),
		LockAcquisitions: acquisitions,
		LockContentions:  atomic.LoadInt64(&l.metrics.lockContentions),
		AverageWaitTime:  avgWaitTime,
		MaxWaitTime:      time.Duration(atomic.LoadInt64(&l.metrics.maxWaitTime)),
	}
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
			atomic.AddInt64(&l.metrics.activeLocks, -1)
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
			atomic.AddInt64(&l.metrics.activeLocks, -1)
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
		atomic.AddInt64(&lw.lock.metrics.lockContentions, 1)
		if lw.lock.opts.OnLockContention != nil {
			lw.lock.opts.OnLockContention(lw.key, int(waiters))
		}
	}

	lw.entry.locker.Lock()

	atomic.AddInt64(&lw.entry.waiters, -1)
	atomic.AddInt64(&lw.lock.metrics.lockAcquisitions, 1)

	// Track wait time
	waitTime := time.Since(lw.startTime)
	atomic.AddInt64(&lw.lock.metrics.totalWaitTime, int64(waitTime))

	// Update max wait time
	for {
		current := atomic.LoadInt64(&lw.lock.metrics.maxWaitTime)
		if int64(waitTime) <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&lw.lock.metrics.maxWaitTime, current, int64(waitTime)) {
			break
		}
	}

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
