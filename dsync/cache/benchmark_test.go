package cache_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

func newBenchClient(b *testing.B) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	b.Helper()
	b.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}

func BenchmarkCache(b *testing.B) {
	client := newBenchClient(b)
	c := cache.New(client)
	ctx := context.Background()

	b.Run("Store", func(b *testing.B) {
		value := []byte("benchmark value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:store:%d", i)
			err := c.Store(ctx, key, value, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Load", func(b *testing.B) {
		// Pre-populate cache
		value := []byte("benchmark value")
		keys := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:load:%d", i)
			keys[i] = key
			err := c.Store(ctx, key, value, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.Load(ctx, keys[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LoadOrStore", func(b *testing.B) {
		value := []byte("benchmark value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:loadorstore:%d", i%100) // Reuse keys to test both paths
			_, _, err := c.LoadOrStore(ctx, key, value, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CompareAndSwap", func(b *testing.B) {
		// Pre-populate cache
		oldValue := []byte("old value")
		newValue := []byte("new value")
		keys := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:cas:%d", i)
			keys[i] = key
			err := c.Store(ctx, key, oldValue, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.CompareAndSwap(ctx, keys[i], oldValue, newValue, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkJSONCache(b *testing.B) {
	client := newBenchClient(b)
	jsonCache := cache.NewJSON(client)
	ctx := context.Background()

	type BenchmarkData struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Data  string `json:"data"`
	}

	data := &BenchmarkData{
		ID:    123,
		Name:  "Benchmark User",
		Email: "bench@example.com",
		Data:  "Some longer data string for more realistic benchmarking",
	}

	b.Run("JSONStore", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:json:store:%d", i)
			err := jsonCache.Store(ctx, key, data, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("JSONLoad", func(b *testing.B) {
		// Pre-populate cache
		keys := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench:json:load:%d", i)
			keys[i] = key
			err := jsonCache.Store(ctx, key, data, time.Hour)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result BenchmarkData
			err := jsonCache.Load(ctx, keys[i], &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("JSONLoadOrStore", func(b *testing.B) {
		getter := func(ctx context.Context, key fmt.Stringer) (*cache.Item, error) {
			return &cache.Item{
				Key:   key.String(),
				TTL:   time.Hour,
				Value: data,
			}, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := cache.PrefixKey{Prefix: "bench:json:loadorstore", Key: strconv.Itoa(i % 100)}
			var result BenchmarkData
			_, err := jsonCache.LoadOrStore(ctx, key, &result, getter)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentAccess tests performance under concurrent load
func BenchmarkConcurrentAccess(b *testing.B) {
	client := newBenchClient(b)
	c := cache.New(client)
	ctx := context.Background()

	// Pre-populate some data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("concurrent:%d", i)
		value := []byte(fmt.Sprintf("value:%d", i))
		err := c.Store(ctx, key, value, time.Hour)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Run("ConcurrentRead", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent:%d", i%100)
				_, err := c.Load(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("ConcurrentWrite", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("concurrent:write:%d", i)
				value := []byte(fmt.Sprintf("value:%d", i))
				err := c.Store(ctx, key, value, time.Hour)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("ConcurrentMixed", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%3 == 0 {
					// Write operation
					key := fmt.Sprintf("concurrent:mixed:%d", i)
					value := []byte(fmt.Sprintf("value:%d", i))
					err := c.Store(ctx, key, value, time.Hour)
					if err != nil {
						b.Fatal(err)
					}
				} else {
					// Read operation
					key := fmt.Sprintf("concurrent:%d", i%100)
					_, err := c.Load(ctx, key)
					if err != nil {
						b.Fatal(err)
					}
				}
				i++
			}
		})
	})
}
