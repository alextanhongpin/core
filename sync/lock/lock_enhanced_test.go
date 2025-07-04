package lock_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

func TestLockBasicFunctionality(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Test basic locking
	locker := l.Get("test-key")
	locker.Lock()
	locker.Unlock()
}

func TestLockWithOptions(t *testing.T) {
	opts := lock.Options{
		DefaultTimeout:  10 * time.Second,
		CleanupInterval: 1 * time.Second,
		LockType:        lock.Mutex,
		MaxLocks:        100,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	locker := l.Get("test-key")
	locker.Lock()
	locker.Unlock()
}

func TestLockWithTimeout(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Test successful lock acquisition
	unlock, err := l.LockWithTimeout("test-key", 1*time.Second)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	unlock()

	// Test timeout
	locker := l.Get("test-key")
	locker.Lock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		locker.Unlock()
	}()

	start := time.Now()
	unlock, err = l.LockWithTimeout("test-key", 50*time.Millisecond)
	elapsed := time.Since(start)

	if err != lock.ErrTimeout {
		t.Fatalf("Expected timeout error, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Timeout took too long: %v", elapsed)
	}
}

func TestLockWithContext(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Test successful lock acquisition
	ctx := context.Background()
	unlock, err := l.LockWithContext(ctx, "test-key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	unlock()

	// Test context cancellation
	locker := l.Get("test-key")
	locker.Lock()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = l.LockWithContext(ctx, "test-key")
	elapsed := time.Since(start)

	if err != lock.ErrCanceled {
		t.Fatalf("Expected canceled error, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Cancellation took too long: %v", elapsed)
	}

	locker.Unlock()
}

func TestLockWithContextTimeout(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Test context timeout
	locker := l.Get("test-key")
	locker.Lock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := l.LockWithContext(ctx, "test-key")
	elapsed := time.Since(start)

	if err != lock.ErrTimeout {
		t.Fatalf("Expected timeout error, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Timeout took too long: %v", elapsed)
	}

	locker.Unlock()
}

func TestRWMutexLock(t *testing.T) {
	opts := lock.Options{
		LockType: lock.RWMutex,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	// Test RWMutex functionality
	rwMutex := l.GetRW("test-key")

	// Test read lock
	rwMutex.RLock()
	rwMutex.RUnlock()

	// Test write lock
	rwMutex.Lock()
	rwMutex.Unlock()
}

func TestRWMutexPanicOnMutexType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic when using GetRW with Mutex type")
		}
	}()

	l := lock.New() // Default is Mutex
	defer l.Stop()

	l.GetRW("test-key") // Should panic
}

func TestLockMetrics(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Initial metrics should be zero
	metrics := l.Metrics()
	if metrics.TotalLocks != 0 || metrics.ActiveLocks != 0 || metrics.LockAcquisitions != 0 {
		t.Fatalf("Initial metrics should be zero, got %+v", metrics)
	}

	// Create some locks
	locker1 := l.Get("key1")
	locker2 := l.Get("key2")

	metrics = l.Metrics()
	if metrics.TotalLocks != 2 || metrics.ActiveLocks != 2 {
		t.Fatalf("Expected 2 locks, got %+v", metrics)
	}

	// Acquire locks
	locker1.Lock()
	locker2.Lock()

	metrics = l.Metrics()
	if metrics.LockAcquisitions != 2 {
		t.Fatalf("Expected 2 acquisitions, got %+v", metrics)
	}

	locker1.Unlock()
	locker2.Unlock()
}

func TestLockContention(t *testing.T) {
	var contentions int64
	var maxWaiters int64

	opts := lock.Options{
		OnLockContention: func(key string, waitingGoroutines int) {
			atomic.AddInt64(&contentions, 1)
			if int64(waitingGoroutines) > atomic.LoadInt64(&maxWaiters) {
				atomic.StoreInt64(&maxWaiters, int64(waitingGoroutines))
			}
		},
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			locker := l.Get("contention-key")
			locker.Lock()
			time.Sleep(10 * time.Millisecond)
			locker.Unlock()
		}()
	}

	wg.Wait()

	if atomic.LoadInt64(&contentions) == 0 {
		t.Fatalf("Expected contention to be detected")
	}
	if atomic.LoadInt64(&maxWaiters) < 2 {
		t.Fatalf("Expected at least 2 waiters, got %d", atomic.LoadInt64(&maxWaiters))
	}

	metrics := l.Metrics()
	if metrics.LockContentions == 0 {
		t.Fatalf("Expected contention metrics to be recorded")
	}
}

func TestLockCallbacks(t *testing.T) {
	var acquisitions int64
	var releases int64
	var totalWaitTime time.Duration
	var totalHoldTime time.Duration

	opts := lock.Options{
		OnLockAcquired: func(key string, waitTime time.Duration) {
			atomic.AddInt64(&acquisitions, 1)
			totalWaitTime += waitTime
		},
		OnLockReleased: func(key string, holdTime time.Duration) {
			atomic.AddInt64(&releases, 1)
			totalHoldTime += holdTime
		},
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	locker := l.Get("callback-key")
	locker.Lock()
	time.Sleep(10 * time.Millisecond)
	locker.Unlock()

	if atomic.LoadInt64(&acquisitions) != 1 {
		t.Fatalf("Expected 1 acquisition, got %d", atomic.LoadInt64(&acquisitions))
	}
	if atomic.LoadInt64(&releases) != 1 {
		t.Fatalf("Expected 1 release, got %d", atomic.LoadInt64(&releases))
	}
	if totalHoldTime < 10*time.Millisecond {
		t.Fatalf("Expected hold time >= 10ms, got %v", totalHoldTime)
	}
}

func TestLockCleanup(t *testing.T) {
	opts := lock.Options{
		CleanupInterval: 100 * time.Millisecond,
		MaxLocks:        5,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	// Create more locks than MaxLocks
	for i := 0; i < 10; i++ {
		locker := l.Get(fmt.Sprintf("cleanup-key-%d", i))
		locker.Lock()
		locker.Unlock()
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check that cleanup occurred
	if size := l.Size(); size > opts.MaxLocks {
		t.Fatalf("Expected size <= %d after cleanup, got %d", opts.MaxLocks, size)
	}
}

func TestLockSize(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	if size := l.Size(); size != 0 {
		t.Fatalf("Expected initial size 0, got %d", size)
	}

	// Create some locks
	l.Get("key1")
	l.Get("key2")
	l.Get("key3")

	if size := l.Size(); size != 3 {
		t.Fatalf("Expected size 3, got %d", size)
	}
}

func TestLockConcurrency(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	const numGoroutines = 100
	const numOperations = 10
	var counter int64

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%10) // 10 different keys

			for j := 0; j < numOperations; j++ {
				locker := l.Get(key)
				locker.Lock()
				atomic.AddInt64(&counter, 1)
				locker.Unlock()
			}
		}(i)
	}

	wg.Wait()

	expected := int64(numGoroutines * numOperations)
	if atomic.LoadInt64(&counter) != expected {
		t.Fatalf("Expected counter %d, got %d", expected, atomic.LoadInt64(&counter))
	}
}

func TestLockInvalidKey(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Test panic on empty key
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic for empty key")
		}
	}()

	l.Get("")
}

func TestLockWithTimeoutInvalidKey(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	unlock, err := l.LockWithTimeout("", 1*time.Second)
	if err != lock.ErrInvalidKey {
		t.Fatalf("Expected invalid key error, got %v", err)
	}
	if unlock != nil {
		t.Fatalf("Expected nil unlock function")
	}
}

func TestLockWithContextInvalidKey(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	ctx := context.Background()
	unlock, err := l.LockWithContext(ctx, "")
	if err != lock.ErrInvalidKey {
		t.Fatalf("Expected invalid key error, got %v", err)
	}
	if unlock != nil {
		t.Fatalf("Expected nil unlock function")
	}
}

func TestLockStop(t *testing.T) {
	l := lock.New()

	// Create some locks
	l.Get("key1")
	l.Get("key2")

	// Stop should not panic
	l.Stop()

	// Multiple stops should not panic
	l.Stop()
	l.Stop()
}

func TestLockWaitTimeTracking(t *testing.T) {
	l := lock.New()
	defer l.Stop()

	// Create contention to generate wait time
	locker := l.Get("wait-key")
	locker.Lock()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		locker.Unlock()
	}()

	// This should wait and generate wait time metrics
	start := time.Now()
	locker2 := l.Get("wait-key")
	locker2.Lock()
	elapsed := time.Since(start)
	locker2.Unlock()

	wg.Wait()

	metrics := l.Metrics()
	if metrics.AverageWaitTime <= 0 {
		t.Fatalf("Expected average wait time > 0, got %v", metrics.AverageWaitTime)
	}
	if metrics.MaxWaitTime <= 0 {
		t.Fatalf("Expected max wait time > 0, got %v", metrics.MaxWaitTime)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("Expected wait time >= 40ms, got %v", elapsed)
	}
}

func TestLockRWMutexConcurrency(t *testing.T) {
	opts := lock.Options{
		LockType: lock.RWMutex,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	const numReaders = 10
	const numWriters = 2
	var readCounter int64
	var writeCounter int64

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			rwMutex := l.GetRW("rw-key")
			for j := 0; j < 10; j++ {
				rwMutex.RLock()
				atomic.AddInt64(&readCounter, 1)
				time.Sleep(1 * time.Millisecond)
				rwMutex.RUnlock()
			}
		}()
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			rwMutex := l.GetRW("rw-key")
			for j := 0; j < 5; j++ {
				rwMutex.Lock()
				atomic.AddInt64(&writeCounter, 1)
				time.Sleep(2 * time.Millisecond)
				rwMutex.Unlock()
			}
		}()
	}

	wg.Wait()

	expectedReads := int64(numReaders * 10)
	expectedWrites := int64(numWriters * 5)

	if atomic.LoadInt64(&readCounter) != expectedReads {
		t.Fatalf("Expected %d reads, got %d", expectedReads, atomic.LoadInt64(&readCounter))
	}
	if atomic.LoadInt64(&writeCounter) != expectedWrites {
		t.Fatalf("Expected %d writes, got %d", expectedWrites, atomic.LoadInt64(&writeCounter))
	}
}
