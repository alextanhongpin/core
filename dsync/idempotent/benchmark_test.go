package idempotent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

func newBenchClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
}

func BenchmarkHandler_Handle(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	fn := func(ctx context.Context, req string) (string, error) {
		return "response", nil
	}

	h := idempotent.NewHandler(client, fn, nil)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-" + string(rune(i%10)) // Use 10 different keys
			_, _, err := h.Handle(ctx, key, "request")
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkHandler_HandleSameKey(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	fn := func(ctx context.Context, req string) (string, error) {
		return "response", nil
	}

	h := idempotent.NewHandler(client, fn, nil)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := h.Handle(ctx, "same-key", "request")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkHandler_ConcurrentSameKey(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	fn := func(ctx context.Context, req string) (string, error) {
		time.Sleep(1 * time.Millisecond) // Simulate some work
		return "response", nil
	}

	h := idempotent.NewHandler(client, fn, nil)
	ctx := context.Background()

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := h.Handle(ctx, "concurrent-key", "request")
			if err != nil {
				b.Error(err)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkRedisStore_Do(b *testing.B) {
	client := newBenchClient()
	defer client.Close()
	store := idempotent.NewRedisStore(client)

	fn := func(ctx context.Context, req []byte) ([]byte, error) {
		return []byte("response"), nil
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-store-" + string(rune(i%10))
			_, _, err := store.Do(ctx, key, fn, []byte("request"), time.Minute, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	fn := func(ctx context.Context, req string) (string, error) {
		return "response", nil
	}

	h := idempotent.NewHandler(client, fn, nil)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := "memory-test-" + string(rune(i%100))
		_, _, err := h.Handle(ctx, key, "request")
		if err != nil {
			b.Fatal(err)
		}
	}
}
