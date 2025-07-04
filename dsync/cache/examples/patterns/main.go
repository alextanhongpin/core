// Package main demonstrates cache with distributed locking patterns.
// This example shows how to implement cache patterns that require
// coordination between multiple instances.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/redis/go-redis/v9"
)

// CacheWithLock demonstrates cache patterns with distributed coordination
type CacheWithLock struct {
	cache     *cache.JSON
	basicCache *cache.Cache
}

func NewCacheWithLock(client *redis.Client) *CacheWithLock {
	return &CacheWithLock{
		cache:     cache.NewJSON(client),
		basicCache: cache.New(client),
	}
}

// ExpensiveComputation simulates an expensive operation that should
// only be executed by one instance at a time
func (c *CacheWithLock) ExpensiveComputation(ctx context.Context, key string) (string, error) {
	lockKey := "lock:" + key
	lockValue := fmt.Sprintf("locked-by-%d", time.Now().UnixNano())
	lockTTL := 30 * time.Second
	
	// Try to acquire lock using StoreOnce (atomic operation)
	err := c.basicCache.StoreOnce(ctx, lockKey, []byte(lockValue), lockTTL)
	if errors.Is(err, cache.ErrExists) {
		// Someone else is already computing, wait and try to get result
		log.Printf("Lock exists for %s, waiting for result...", key)
		return c.waitForResult(ctx, key, 5*time.Second)
	}
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	log.Printf("Acquired lock for %s, computing...", key)
	
	// We have the lock, ensure we release it
	defer func() {
		// Use CompareAndDelete to safely release the lock
		if err := c.basicCache.CompareAndDelete(ctx, lockKey, []byte(lockValue)); err != nil {
			log.Printf("Failed to release lock for %s: %v", key, err)
		} else {
			log.Printf("Released lock for %s", key)
		}
	}()
	
	// Simulate expensive computation
	time.Sleep(2 * time.Second)
	result := fmt.Sprintf("computed-result-for-%s-at-%d", key, time.Now().Unix())
	
	// Store the result
	err = c.basicCache.Store(ctx, key, []byte(result), 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to store result: %w", err)
	}
	
	log.Printf("Stored result for %s", key)
	return result, nil
}

func (c *CacheWithLock) waitForResult(ctx context.Context, key string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Check if result is available
		result, err := c.basicCache.Load(ctx, key)
		if err == nil {
			log.Printf("Got result after waiting for %s", key)
			return string(result), nil
		}
		
		if !errors.Is(err, cache.ErrNotExist) {
			return "", err
		}
		
		// Wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}
	
	return "", errors.New("timeout waiting for result")
}

// CacheWriteThrough demonstrates write-through caching pattern
func (c *CacheWithLock) CacheWriteThrough(ctx context.Context, key string, value any, persistFunc func() error) error {
	// First persist to the primary store
	if err := persistFunc(); err != nil {
		return fmt.Errorf("failed to persist: %w", err)
	}
	
	// Then update cache
	if err := c.cache.Store(ctx, key, value, time.Hour); err != nil {
		log.Printf("Warning: failed to update cache for %s: %v", key, err)
		// Don't fail the operation if cache update fails
	}
	
	return nil
}

// CacheWriteBehind demonstrates write-behind (write-back) caching pattern
func (c *CacheWithLock) CacheWriteBehind(ctx context.Context, key string, value any) error {
	// First update cache immediately
	if err := c.cache.Store(ctx, key, value, time.Hour); err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}
	
	// Mark for async persistence (in real world, you'd queue this)
	dirtyKey := "dirty:" + key
	if err := c.basicCache.Store(ctx, dirtyKey, []byte("1"), time.Hour); err != nil {
		log.Printf("Warning: failed to mark %s as dirty: %v", key, err)
	}
	
	log.Printf("Cached %s and marked for async persistence", key)
	return nil
}

// RefreshAhead demonstrates refresh-ahead caching pattern
func (c *CacheWithLock) RefreshAhead(ctx context.Context, key string, refreshFunc func() (any, error), refreshThreshold time.Duration) (any, error) {
	// Check TTL of the key
	ttl, err := c.cache.TTL(ctx, key)
	if err != nil && !errors.Is(err, cache.ErrNotExist) {
		return nil, fmt.Errorf("failed to get TTL: %w", err)
	}
	
	var result any
	
	// If key doesn't exist or TTL is below threshold, refresh in background
	if errors.Is(err, cache.ErrNotExist) || (ttl > 0 && ttl < refreshThreshold) {
		log.Printf("Key %s needs refresh (TTL: %v)", key, ttl)
		
		// Start async refresh
		go func() {
			refreshCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			refreshKey := "refresh:" + key
			
			// Use atomic operation to ensure only one refresh happens
			err := c.basicCache.StoreOnce(refreshCtx, refreshKey, []byte("refreshing"), 1*time.Minute)
			if errors.Is(err, cache.ErrExists) {
				log.Printf("Refresh already in progress for %s", key)
				return
			}
			
			defer c.basicCache.Delete(refreshCtx, refreshKey)
			
			log.Printf("Starting background refresh for %s", key)
			newValue, err := refreshFunc()
			if err != nil {
				log.Printf("Failed to refresh %s: %v", key, err)
				return
			}
			
			if err := c.cache.Store(refreshCtx, key, newValue, time.Hour); err != nil {
				log.Printf("Failed to store refreshed value for %s: %v", key, err)
			} else {
				log.Printf("Successfully refreshed %s", key)
			}
		}()
	}
	
	// Try to load current value
	err = c.cache.Load(ctx, key, &result)
	if errors.Is(err, cache.ErrNotExist) {
		// No cached value, refresh synchronously
		log.Printf("No cached value for %s, refreshing synchronously", key)
		result, err = refreshFunc()
		if err != nil {
			return nil, err
		}
		
		// Store the result
		if err := c.cache.Store(ctx, key, result, time.Hour); err != nil {
			log.Printf("Warning: failed to cache result for %s: %v", key, err)
		}
	} else if err != nil {
		return nil, err
	}
	
	return result, nil
}

func main() {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	defer client.Close()
	
	// Flush Redis for clean demo
	client.FlushAll(context.Background()).Err()
	
	cacheSystem := NewCacheWithLock(client)
	ctx := context.Background()
	
	fmt.Println("=== Advanced Cache Patterns Demo ===\n")
	
	// Demonstrate distributed locking with expensive computation
	demonstrateDistributedLocking(ctx, cacheSystem)
	
	// Demonstrate write patterns
	demonstrateWritePatterns(ctx, cacheSystem)
	
	// Demonstrate refresh-ahead pattern
	demonstrateRefreshAhead(ctx, cacheSystem)
}

func demonstrateDistributedLocking(ctx context.Context, cacheSystem *CacheWithLock) {
	fmt.Println("1. Distributed Locking Pattern")
	fmt.Println("Simulating multiple concurrent requests for expensive computation...\n")
	
	var wg sync.WaitGroup
	key := "expensive-result"
	
	// Simulate 5 concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			log.Printf("Worker %d: Starting expensive computation request", id)
			result, err := cacheSystem.ExpensiveComputation(ctx, key)
			if err != nil {
				log.Printf("Worker %d: Error: %v", id, err)
			} else {
				log.Printf("Worker %d: Got result: %s", id, result)
			}
		}(i)
	}
	
	wg.Wait()
	fmt.Println("\nDistributed locking demo complete.\n")
}

func demonstrateWritePatterns(ctx context.Context, cacheSystem *CacheWithLock) {
	fmt.Println("2. Write Patterns")
	
	// Write-through example
	fmt.Println("Write-through pattern:")
	data := map[string]string{"name": "John", "email": "john@example.com"}
	err := cacheSystem.CacheWriteThrough(ctx, "user:1", data, func() error {
		log.Println("Persisting to primary database...")
		time.Sleep(50 * time.Millisecond) // Simulate DB write
		return nil
	})
	if err != nil {
		log.Printf("Write-through failed: %v", err)
	}
	
	// Write-behind example
	fmt.Println("\nWrite-behind pattern:")
	data2 := map[string]string{"name": "Jane", "email": "jane@example.com"}
	err = cacheSystem.CacheWriteBehind(ctx, "user:2", data2)
	if err != nil {
		log.Printf("Write-behind failed: %v", err)
	}
	
	fmt.Println("Write patterns demo complete.\n")
}

func demonstrateRefreshAhead(ctx context.Context, cacheSystem *CacheWithLock) {
	fmt.Println("3. Refresh-ahead Pattern")
	
	key := "dynamic-content"
	refreshFunc := func() (any, error) {
		log.Println("Executing refresh function (expensive operation)...")
		time.Sleep(500 * time.Millisecond) // Simulate expensive operation
		return map[string]any{
			"content":    "Fresh content",
			"timestamp":  time.Now().Unix(),
			"version":    "1.0",
		}, nil
	}
	
	// First call - will cache the result
	fmt.Println("First call (cache miss):")
	result, err := cacheSystem.RefreshAhead(ctx, key, refreshFunc, 30*time.Second)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: %+v\n", result)
	}
	
	// Second call - will return cached result
	fmt.Println("\nSecond call (cache hit):")
	result, err = cacheSystem.RefreshAhead(ctx, key, refreshFunc, 30*time.Second)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: %+v\n", result)
	}
	
	fmt.Println("\nRefresh-ahead demo complete.")
	fmt.Println("\n=== All Demos Complete ===")
}
