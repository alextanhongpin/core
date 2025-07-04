package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/sync/batch"
)

// User represents a user entity
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	ctx := context.Background()

	// Demonstrate advanced batch loader with metrics and callbacks
	fmt.Println("=== Advanced Batch Loader Demo ===")

	var batchCallCount int
	opts := batch.LoaderOptions[int, User]{
		BatchFn: func(keys []int) (map[int]User, error) {
			batchCallCount++
			fmt.Printf("📦 Batch call #%d: Loading users %v\n", batchCallCount, keys)

			// Simulate database query with delay
			time.Sleep(10 * time.Millisecond)

			users := make(map[int]User)
			for _, id := range keys {
				users[id] = User{
					ID:   id,
					Name: fmt.Sprintf("User %d", id),
				}
			}

			return users, nil
		},
		TTL:          time.Hour,
		MaxBatchSize: 10,
		OnBatchCall: func(keys []int, duration time.Duration, err error) {
			fmt.Printf("⏱️  Batch processed %d keys in %v (error: %v)\n", len(keys), duration, err)
		},
		OnCacheHit: func(keys []int) {
			fmt.Printf("✅ Cache hit for keys: %v\n", keys)
		},
		OnCacheMiss: func(keys []int) {
			fmt.Printf("❌ Cache miss for keys: %v\n", keys)
		},
	}

	loader := batch.NewLoader[int, User](&opts)

	// First batch - all cache misses
	fmt.Println("\n--- First batch (cache misses) ---")
	users, err := loader.LoadMany(ctx, []int{1, 2, 3})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded users: %+v\n", users)

	// Second batch - mix of hits and misses
	fmt.Println("\n--- Second batch (mixed hits/misses) ---")
	users, err = loader.LoadMany(ctx, []int{2, 3, 4, 5}) // 2,3 should be cache hits
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded users: %+v\n", users)

	// Show metrics
	fmt.Println("\n--- Loader Metrics ---")
	metrics := loader.Metrics()
	fmt.Printf("📊 Cache hits: %d\n", metrics.CacheHits)
	fmt.Printf("📊 Cache misses: %d\n", metrics.CacheMisses)
	fmt.Printf("📊 Batch calls: %d\n", metrics.BatchCalls)
	fmt.Printf("📊 Total keys processed: %d\n", metrics.TotalKeys)
	fmt.Printf("📊 Errors: %d\n", metrics.ErrorCount)

	// Demonstrate cache directly
	fmt.Println("\n=== Cache Demo ===")
	cache := batch.NewCache[string, string]()

	// Store some data
	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err = cache.StoreMany(ctx, data, time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	// Load data
	result, err := cache.LoadMany(ctx, "key1", "key2", "key4") // key4 doesn't exist
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cache result: %+v\n", result)

	// Show cache metrics
	fmt.Println("\n--- Cache Metrics ---")
	cacheMetrics := cache.Metrics()
	fmt.Printf("📊 Gets: %d\n", cacheMetrics.Gets)
	fmt.Printf("📊 Sets: %d\n", cacheMetrics.Sets)
	fmt.Printf("📊 Hits: %d\n", cacheMetrics.Hits)
	fmt.Printf("📊 Misses: %d\n", cacheMetrics.Misses)
	fmt.Printf("📊 Size: %d\n", cacheMetrics.Size)
	fmt.Printf("📊 Evictions: %d\n", cacheMetrics.Evictions)

	// Demonstrate TTL expiration
	fmt.Println("\n=== TTL Demo ===")
	shortTTLCache := batch.NewCache[string, string]()

	err = shortTTLCache.StoreMany(ctx, map[string]string{
		"temp1": "value1",
		"temp2": "value2",
	}, 50*time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}

	// Load immediately
	result, err = shortTTLCache.LoadMany(ctx, "temp1", "temp2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Before expiration: %+v\n", result)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Try to load again
	result, err = shortTTLCache.LoadMany(ctx, "temp1", "temp2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After expiration: %+v\n", result)

	// Clean up expired entries
	shortTTLCache.CleanupExpired()

	fmt.Println("\n=== Demo Complete ===")
}
