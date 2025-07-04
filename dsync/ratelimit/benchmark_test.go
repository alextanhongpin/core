package ratelimit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func newBenchClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
}

func BenchmarkFixedWindow_Allow(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 1000000, time.Minute) // High limit
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-fw-" + string(rune(i%10)) // 10 different keys
			_, err := rl.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkFixedWindow_AllowN(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 1000000, time.Minute)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-fw-n-" + string(rune(i%10))
			_, err := rl.AllowN(ctx, key, 5)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkFixedWindow_Check(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 1000000, time.Minute)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-fw-check-" + string(rune(i%10))
			_, err := rl.Check(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkGCRA_Allow(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 1000000, time.Minute, 1000) // High limit
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-gcra-" + string(rune(i%10))
			_, err := rl.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkGCRA_AllowN(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 1000000, time.Minute, 1000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-gcra-n-" + string(rune(i%10))
			_, err := rl.AllowN(ctx, key, 5)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkGCRA_Check(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 1000000, time.Minute, 1000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-gcra-check-" + string(rune(i%10))
			_, err := rl.Check(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// Benchmark comparing Fixed Window vs GCRA
func BenchmarkComparison(b *testing.B) {
	client := newBenchClient()
	defer client.Close()
	ctx := context.Background()

	b.Run("FixedWindow", func(b *testing.B) {
		rl := ratelimit.NewFixedWindow(client, 100000, time.Minute)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "comp-fw-" + string(rune(i%5))
				_, err := rl.Allow(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("GCRA", func(b *testing.B) {
		rl := ratelimit.NewGCRA(client, 100000, time.Minute, 100)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "comp-gcra-" + string(rune(i%5))
				_, err := rl.Allow(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}

// Memory allocation benchmark
func BenchmarkFixedWindow_Memory(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 1000, time.Minute)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "bench-fw-mem-" + string(rune(i%100))
		_, err := rl.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGCRA_Memory(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 1000, time.Minute, 10)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "bench-gcra-mem-" + string(rune(i%100))
		_, err := rl.Allow(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// High contention benchmark (same key)
func BenchmarkFixedWindow_HighContention(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 1000000, time.Minute)
	ctx := context.Background()
	key := "high-contention-fw"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGCRA_HighContention(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 1000000, time.Minute, 1000)
	ctx := context.Background()
	key := "high-contention-gcra"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rl.Allow(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Burst handling benchmark
func BenchmarkFixedWindow_Burst(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewFixedWindow(client, 100, time.Second)
	ctx := context.Background()

	b.ResetTimer()

	// Simulate burst traffic
	var wg sync.WaitGroup
	errCh := make(chan error, b.N)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "burst-fw-" + string(rune(i%10))
			_, err := rl.Allow(ctx, key)
			if err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()

	select {
	case err := <-errCh:
		b.Fatal(err)
	default:
	}
}

func BenchmarkGCRA_Burst(b *testing.B) {
	client := newBenchClient()
	defer client.Close()

	rl := ratelimit.NewGCRA(client, 100, time.Second, 10)
	ctx := context.Background()

	b.ResetTimer()

	// Simulate burst traffic
	var wg sync.WaitGroup
	errCh := make(chan error, b.N)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "burst-gcra-" + string(rune(i%10))
			_, err := rl.Allow(ctx, key)
			if err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()

	select {
	case err := <-errCh:
		b.Fatal(err)
	default:
	}
}

// Realistic workload benchmark
func BenchmarkRealisticWorkload(b *testing.B) {
	client := newBenchClient()
	defer client.Close()
	ctx := context.Background()

	b.Run("FixedWindow_API", func(b *testing.B) {
		rl := ratelimit.NewFixedWindow(client, 1000, time.Hour) // API quota
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				userKey := "user:" + string(rune(i%100)) // 100 different users
				_, err := rl.Allow(ctx, userKey)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("GCRA_Stream", func(b *testing.B) {
		rl := ratelimit.NewGCRA(client, 100, time.Second, 10) // Stream processing
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				streamKey := "stream:" + string(rune(i%10)) // 10 different streams
				_, err := rl.Allow(ctx, streamKey)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}
