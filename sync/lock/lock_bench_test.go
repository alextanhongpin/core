package lock_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

// BenchmarkLockBasic benchmarks basic lock operations
func BenchmarkLockBasic(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker := l.Get("bench-key")
			locker.Lock()
			locker.Unlock()
		}
	})
}

// BenchmarkLockMultipleKeys benchmarks lock operations with multiple keys
func BenchmarkLockMultipleKeys(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", i%10)
			locker := l.Get(key)
			locker.Lock()
			locker.Unlock()
			i++
		}
	})
}

// BenchmarkLockHighContention benchmarks lock operations under high contention
func BenchmarkLockHighContention(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker := l.Get("contention-key")
			locker.Lock()
			// Simulate some work
			time.Sleep(1 * time.Microsecond)
			locker.Unlock()
		}
	})
}

// BenchmarkLockWithTimeout benchmarks lock operations with timeout
func BenchmarkLockWithTimeout(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("timeout-key-%d", i%10)
			unlock, err := l.LockWithTimeout(key, 1*time.Second)
			if err != nil {
				b.Fatal(err)
			}
			unlock()
			i++
		}
	})
}

// BenchmarkRWMutexRead benchmarks read operations with RWMutex
func BenchmarkRWMutexRead(b *testing.B) {
	opts := lock.Options{
		LockType: lock.RWMutex,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rwMutex := l.GetRW("rw-key")
			rwMutex.RLock()
			rwMutex.RUnlock()
		}
	})
}

// BenchmarkRWMutexWrite benchmarks write operations with RWMutex
func BenchmarkRWMutexWrite(b *testing.B) {
	opts := lock.Options{
		LockType: lock.RWMutex,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rwMutex := l.GetRW("rw-key")
			rwMutex.Lock()
			rwMutex.Unlock()
		}
	})
}

// BenchmarkRWMutexMixed benchmarks mixed read/write operations with RWMutex
func BenchmarkRWMutexMixed(b *testing.B) {
	opts := lock.Options{
		LockType: lock.RWMutex,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			rwMutex := l.GetRW("rw-mixed-key")
			if i%10 == 0 {
				// 10% writes
				rwMutex.Lock()
				rwMutex.Unlock()
			} else {
				// 90% reads
				rwMutex.RLock()
				rwMutex.RUnlock()
			}
			i++
		}
	})
}

// BenchmarkLockWithCallbacks benchmarks lock operations with callbacks
func BenchmarkLockWithCallbacks(b *testing.B) {
	opts := lock.Options{
		OnLockAcquired: func(key string, waitTime time.Duration) {
			// Simulate callback overhead
		},
		OnLockReleased: func(key string, holdTime time.Duration) {
			// Simulate callback overhead
		},
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker := l.Get("callback-key")
			locker.Lock()
			locker.Unlock()
		}
	})
}

// BenchmarkLockMetrics benchmarks the metrics collection overhead
func BenchmarkLockMetrics(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker := l.Get("metrics-key")
			locker.Lock()
			locker.Unlock()
			// Get metrics to measure overhead
			_ = l.Metrics()
		}
	})
}

// BenchmarkStandardMutex benchmarks standard Go mutex for comparison
func BenchmarkStandardMutex(b *testing.B) {
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			mu.Unlock()
		}
	})
}

// BenchmarkStandardRWMutex benchmarks standard Go RWMutex for comparison
func BenchmarkStandardRWMutex(b *testing.B) {
	var mu sync.RWMutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			mu.RUnlock()
		}
	})
}

// BenchmarkLockCreation benchmarks lock creation overhead
func BenchmarkLockCreation(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("creation-key-%d", i)
		_ = l.Get(key)
	}
}

// BenchmarkLockCleanup benchmarks the cleanup operation
func BenchmarkLockCleanup(b *testing.B) {
	opts := lock.Options{
		CleanupInterval: 1 * time.Hour, // Disable automatic cleanup
		MaxLocks:        1000,
	}
	l := lock.NewWithOptions(opts)
	defer l.Stop()

	// Create many locks
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("cleanup-key-%d", i)
		locker := l.Get(key)
		locker.Lock()
		locker.Unlock()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Manually trigger cleanup (access private method via reflection would be needed)
		// For now, we'll just measure the creation overhead
		_ = l.Size()
	}
}

// BenchmarkLockMemoryUsage benchmarks memory usage patterns
func BenchmarkLockMemoryUsage(b *testing.B) {
	l := lock.New()
	defer l.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Create locks with unique keys to test memory usage
			key := fmt.Sprintf("memory-key-%d", i)
			locker := l.Get(key)
			locker.Lock()
			locker.Unlock()
			i++
		}
	})
}

// BenchmarkLockScalability benchmarks lock scalability with increasing keys
func BenchmarkLockScalability(b *testing.B) {
	scales := []int{1, 10, 100, 1000}

	for _, scale := range scales {
		b.Run(fmt.Sprintf("Keys-%d", scale), func(b *testing.B) {
			l := lock.New()
			defer l.Stop()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("scale-key-%d", i%scale)
					locker := l.Get(key)
					locker.Lock()
					locker.Unlock()
					i++
				}
			})
		})
	}
}
