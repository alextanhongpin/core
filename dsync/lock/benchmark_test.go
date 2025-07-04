package lock_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/redis/go-redis/v9"
)

func newTestClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func BenchmarkLock_BasicLocking(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-basic-" + string(rune(i%10)) // 10 different keys
			err := locker.Do(ctx, key, func(ctx context.Context) error {
				// Minimal work
				return nil
			}, &lock.LockOption{
				Lock: time.Second,
				Wait: 0,
			})
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkLock_TryLockUnlock(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-try-" + string(rune(i%10))
			token := "token-" + string(rune(i))

			err := locker.TryLock(ctx, key, token, time.Second)
			if err != nil {
				continue // Skip if locked
			}

			err = locker.Unlock(ctx, key, token)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkLock_WithWaiting(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-wait-" + string(rune(i%5)) // More contention
			err := locker.Do(ctx, key, func(ctx context.Context) error {
				// Minimal work
				return nil
			}, &lock.LockOption{
				Lock: 100 * time.Millisecond,
				Wait: 500 * time.Millisecond,
			})
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkLock_PubSub(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.NewPubSub(client)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-pubsub-" + string(rune(i%5))
			err := locker.Do(ctx, key, func(ctx context.Context) error {
				return nil
			}, &lock.LockOption{
				Lock: 100 * time.Millisecond,
				Wait: 500 * time.Millisecond,
			})
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkLock_HighContention(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	// Single key for maximum contention
	key := "bench-high-contention"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := locker.Do(ctx, key, func(ctx context.Context) error {
				// Minimal work
				return nil
			}, &lock.LockOption{
				Lock: 50 * time.Millisecond,
				Wait: 200 * time.Millisecond,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkLock_Extend(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	// Pre-acquire locks
	tokens := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		key := "bench-extend-" + string(rune(i))
		token := "token-" + string(rune(i))
		err := locker.TryLock(ctx, key, token, time.Minute)
		if err != nil {
			b.Fatal(err)
		}
		tokens[i] = token
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-extend-" + string(rune(i))
		token := tokens[i]
		err := locker.Extend(ctx, key, token, time.Minute)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark comparing basic locker vs PubSub locker
func BenchmarkLock_Comparison(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	b.Run("Basic", func(b *testing.B) {
		locker := lock.New(client)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "bench-comp-basic-" + string(rune(i%3))
				err := locker.Do(ctx, key, func(ctx context.Context) error {
					return nil
				}, &lock.LockOption{
					Lock: 100 * time.Millisecond,
					Wait: 300 * time.Millisecond,
				})
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("PubSub", func(b *testing.B) {
		locker := lock.NewPubSub(client)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "bench-comp-pubsub-" + string(rune(i%3))
				err := locker.Do(ctx, key, func(ctx context.Context) error {
					return nil
				}, &lock.LockOption{
					Lock: 100 * time.Millisecond,
					Wait: 300 * time.Millisecond,
				})
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}

// Memory allocation benchmark
func BenchmarkLock_Memory(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "bench-memory-" + string(rune(i%100))
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			return nil
		}, &lock.LockOption{
			Lock: time.Second,
			Wait: 0,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Concurrent lock operations benchmark
func BenchmarkLock_ConcurrentOperations(b *testing.B) {
	client := newTestClient()
	defer client.Close()
	locker := lock.New(client)
	ctx := context.Background()

	b.ResetTimer()

	var wg sync.WaitGroup
	workers := 10

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < b.N/workers; i++ {
				key := "bench-concurrent-" + string(rune(i%5))
				err := locker.Do(ctx, key, func(ctx context.Context) error {
					// Simulate minimal work
					return nil
				}, &lock.LockOption{
					Lock: 100 * time.Millisecond,
					Wait: 200 * time.Millisecond,
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		}(w)
	}

	wg.Wait()
}
