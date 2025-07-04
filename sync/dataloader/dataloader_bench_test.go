package dataloader

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

func BenchmarkDataLoader(b *testing.B) {
	b.Run("single_load", func(b *testing.B) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 1 * time.Millisecond,
		})
		defer dl.Stop()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := strconv.Itoa(i % 100) // Cycle through 100 keys
				dl.Load(key)
				i++
			}
		})
	})

	b.Run("batch_load", func(b *testing.B) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 10 * time.Millisecond,
		})
		defer dl.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			keys := make([]string, 10)
			for j := 0; j < 10; j++ {
				keys[j] = strconv.Itoa(i*10 + j)
			}
			dl.LoadMany(keys)
		}
	})

	b.Run("cached_load", func(b *testing.B) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 1 * time.Millisecond,
		})
		defer dl.Stop()

		// Pre-populate cache
		for i := 0; i < 100; i++ {
			dl.Load(strconv.Itoa(i))
		}
		time.Sleep(20 * time.Millisecond) // Wait for batches to complete

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := strconv.Itoa(i % 100) // All keys should be cached
				dl.Load(key)
				i++
			}
		})
	})

	b.Run("high_concurrency", func(b *testing.B) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 5 * time.Millisecond,
		})
		defer dl.Stop()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			var wg sync.WaitGroup
			for pb.Next() {
				wg.Add(10)
				for i := 0; i < 10; i++ {
					go func(i int) {
						defer wg.Done()
						key := strconv.Itoa(i)
						dl.Load(key)
					}(i)
				}
				wg.Wait()
			}
		})
	})
}

func BenchmarkDataLoaderVsDirectCall(b *testing.B) {
	batchFn := func(ctx context.Context, keys []string) (map[string]int, error) {
		result := make(map[string]int)
		for _, key := range keys {
			result[key] = len(key)
		}
		return result, nil
	}

	b.Run("direct_call", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := strconv.Itoa(i % 100)
				batchFn(context.Background(), []string{key})
				i++
			}
		})
	})

	b.Run("dataloader", func(b *testing.B) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn:      batchFn,
			BatchTimeout: 1 * time.Millisecond,
		})
		defer dl.Stop()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := strconv.Itoa(i % 100)
				dl.Load(key)
				i++
			}
		})
	})
}

func BenchmarkMetricsAccess(b *testing.B) {
	dl := New(context.Background(), &Options[string, int]{
		BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
			result := make(map[string]int)
			for _, key := range keys {
				result[key] = len(key)
			}
			return result, nil
		},
		BatchTimeout: 1 * time.Millisecond,
	})
	defer dl.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			dl.Metrics()
		}
	})
}
