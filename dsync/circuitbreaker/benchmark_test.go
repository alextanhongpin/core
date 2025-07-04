package circuitbreaker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func newBenchClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
}

func BenchmarkCircuitBreaker_Do_Success(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-success")
	defer stop()

	ctx := context.Background()
	fn := func() error { return nil }

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := cb.Do(ctx, fn)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCircuitBreaker_Do_Failure(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-failure")
	defer stop()

	ctx := context.Background()
	fn := func() error { return wantErr }

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Do(ctx, fn) // Ignore errors for benchmarking
		}
	})
}

func BenchmarkCircuitBreaker_Do_Open(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-open")
	defer stop()

	// Force circuit to open
	ctx := context.Background()
	for i := 0; i < cb.FailureThreshold; i++ {
		cb.Do(ctx, func() error { return wantErr })
	}

	// Wait for circuit to open
	time.Sleep(100 * time.Millisecond)

	fn := func() error { return nil }

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Do(ctx, fn) // Will return ErrUnavailable
		}
	})
}

func BenchmarkCircuitBreaker_Status(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-status")
	defer stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Status()
		}
	})
}

func BenchmarkCircuitBreaker_Concurrent(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-concurrent")
	defer stop()

	ctx := context.Background()
	fn := func() error {
		time.Sleep(1 * time.Millisecond) // Simulate work
		return nil
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Do(ctx, fn)
			if err != nil {
				b.Error(err)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkCircuitBreaker_Memory(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	cb, stop := circuitbreaker.New(client, "bench-memory")
	defer stop()

	ctx := context.Background()
	fn := func() error { return nil }

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := cb.Do(ctx, fn)
		if err != nil {
			b.Fatal(err)
		}
	}
}
