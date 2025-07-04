package batch_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/batch"
)

func BenchmarkLoader(b *testing.B) {
	ctx := context.Background()

	b.Run("basic_loader", func(b *testing.B) {
		var callCount int64

		opts := &batch.LoaderOptions[int, string]{
			BatchFn: func(keys []int) (map[int]string, error) {
				atomic.AddInt64(&callCount, 1)
				m := make(map[int]string)
				for _, k := range keys {
					m[k] = strconv.Itoa(k)
				}
				return m, nil
			},
			TTL: time.Hour,
		}

		loader := batch.NewLoader(opts)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, err := loader.Load(ctx, i%100) // Load from 0-99
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("batch_loader", func(b *testing.B) {
		var callCount int64

		opts := &batch.LoaderOptions[int, string]{
			BatchFn: func(keys []int) (map[int]string, error) {
				atomic.AddInt64(&callCount, 1)
				m := make(map[int]string)
				for _, k := range keys {
					m[k] = strconv.Itoa(k)
				}
				return m, nil
			},
			TTL:          time.Hour,
			MaxBatchSize: 50,
		}

		loader := batch.NewLoader(opts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			keys := make([]int, 10)
			for j := range keys {
				keys[j] = (i*10 + j) % 100
			}
			_, err := loader.LoadMany(ctx, keys)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("with_callbacks", func(b *testing.B) {
		var callCount int64

		opts := &batch.LoaderOptions[int, string]{
			BatchFn: func(keys []int) (map[int]string, error) {
				atomic.AddInt64(&callCount, 1)
				m := make(map[int]string)
				for _, k := range keys {
					m[k] = strconv.Itoa(k)
				}
				return m, nil
			},
			TTL: time.Hour,
			OnCacheHit: func(keys []int) {
				// Track cache hits
			},
			OnCacheMiss: func(keys []int) {
				// Track cache misses
			},
			OnBatchCall: func(keys []int, duration time.Duration, err error) {
				// Track batch calls
			},
		}

		loader := batch.NewLoader(opts)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, err := loader.Load(ctx, i%100)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}

func BenchmarkCache(b *testing.B) {
	ctx := context.Background()

	b.Run("cache_operations", func(b *testing.B) {
		cache := batch.NewCache[int, string]()

		// Pre-populate cache
		data := make(map[int]string)
		for i := 0; i < 1000; i++ {
			data[i] = strconv.Itoa(i)
		}
		cache.StoreMany(ctx, data, time.Hour)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, err := cache.LoadMany(ctx, i%1000)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("cache_metrics", func(b *testing.B) {
		cache := batch.NewCache[int, string]()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Metrics()
			}
		})
	})
}

func BenchmarkLoaderMetrics(b *testing.B) {
	opts := &batch.LoaderOptions[int, string]{
		BatchFn: func(keys []int) (map[int]string, error) {
			m := make(map[int]string)
			for _, k := range keys {
				m[k] = strconv.Itoa(k)
			}
			return m, nil
		},
		TTL: time.Hour,
	}

	loader := batch.NewLoader(opts)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			loader.Metrics()
		}
	})
}
