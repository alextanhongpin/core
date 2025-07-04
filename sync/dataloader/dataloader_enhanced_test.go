package dataloader

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDataLoaderEnhanced(t *testing.T) {
	t.Run("metrics tracking", func(t *testing.T) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					if key == "error" {
						continue // This will cause ErrNoResult for this key
					}
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 10 * time.Millisecond,
		})
		defer dl.Stop()

		// Test successful loads
		_, err := dl.Load("hello")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Test cache hit
		_, err = dl.Load("hello")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Test error case
		_, err = dl.Load("error")
		if err == nil {
			t.Fatal("Expected error for 'error' key")
		}

		// Check metrics
		metrics := dl.Metrics()
		if metrics.TotalRequests != 3 {
			t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
		}
		if metrics.CacheHits != 1 {
			t.Errorf("Expected 1 cache hit, got %d", metrics.CacheHits)
		}
		if metrics.CacheMisses != 2 {
			t.Errorf("Expected 2 cache misses, got %d", metrics.CacheMisses)
		}
		if metrics.NoResultCount != 1 {
			t.Errorf("Expected 1 no result, got %d", metrics.NoResultCount)
		}
	})

	t.Run("callbacks", func(t *testing.T) {
		var callbackCalls []string
		var mu sync.Mutex

		addCall := func(call string) {
			mu.Lock()
			callbackCalls = append(callbackCalls, call)
			mu.Unlock()
		}

		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 10 * time.Millisecond,
			OnBatchStart: func(keys []string) {
				addCall("batch_start")
			},
			OnBatchComplete: func(keys []string, duration time.Duration, err error) {
				addCall("batch_complete")
			},
			OnCacheHit: func(key string) {
				addCall("cache_hit")
			},
			OnCacheMiss: func(key string) {
				addCall("cache_miss")
			},
		})
		defer dl.Stop()

		// First load - should trigger batch
		_, err := dl.Load("test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Wait for batch to complete
		time.Sleep(20 * time.Millisecond)

		// Second load - should hit cache
		_, err = dl.Load("test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		expectedCalls := []string{"cache_miss", "batch_start", "batch_complete", "cache_hit"}
		if len(callbackCalls) != len(expectedCalls) {
			t.Errorf("Expected %d callback calls, got %d: %v", len(expectedCalls), len(callbackCalls), callbackCalls)
		}
	})

	t.Run("load with timeout", func(t *testing.T) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				// Simulate slow operation
				time.Sleep(100 * time.Millisecond)
				result := make(map[string]int)
				for _, key := range keys {
					result[key] = len(key)
				}
				return result, nil
			},
			BatchTimeout: 10 * time.Millisecond,
			LoadTimeout:  50 * time.Millisecond,
		})
		defer dl.Stop()

		// This should timeout
		_, err := dl.LoadWithTimeout(context.Background(), "test")
		if err == nil {
			t.Fatal("Expected timeout error")
		}
		if !errors.Is(err, ErrTimeout) {
			t.Errorf("Expected ErrTimeout, got %v", err)
		}
	})

	t.Run("cache operations", func(t *testing.T) {
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

		// Load some data
		_, err := dl.Load("test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Wait for batch
		time.Sleep(20 * time.Millisecond)

		// Check cache size
		if size := dl.opts.Cache.(*Cache[string, int]).Size(); size != 1 {
			t.Errorf("Expected cache size 1, got %d", size)
		}

		// Set a value directly
		dl.Set("manual", 42)
		if size := dl.opts.Cache.(*Cache[string, int]).Size(); size != 2 {
			t.Errorf("Expected cache size 2, got %d", size)
		}

		// Clear cache
		dl.ClearCache()
		if size := dl.opts.Cache.(*Cache[string, int]).Size(); size != 0 {
			t.Errorf("Expected cache size 0 after clear, got %d", size)
		}
	})

	t.Run("concurrent load many", func(t *testing.T) {
		dl := New(context.Background(), &Options[int, string]{
			BatchFn: func(ctx context.Context, keys []int) (map[int]string, error) {
				result := make(map[int]string)
				for _, key := range keys {
					result[key] = string(rune('A' + key))
				}
				return result, nil
			},
			BatchTimeout: 10 * time.Millisecond,
		})
		defer dl.Stop()

		// Load many keys concurrently
		var wg sync.WaitGroup
		results := make([]string, 10)
		errors := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i], errors[i] = dl.Load(i)
			}(i)
		}

		wg.Wait()

		// Check results
		for i, err := range errors {
			if err != nil {
				t.Errorf("Expected no error for key %d, got %v", i, err)
			}
		}

		for i, result := range results {
			expected := string(rune('A' + i))
			if result != expected {
				t.Errorf("Expected result %s for key %d, got %s", expected, i, result)
			}
		}
	})

	t.Run("error handling", func(t *testing.T) {
		dl := New(context.Background(), &Options[string, int]{
			BatchFn: func(ctx context.Context, keys []string) (map[string]int, error) {
				return nil, errors.New("batch function error")
			},
			BatchTimeout: 10 * time.Millisecond,
		})
		defer dl.Stop()

		_, err := dl.Load("test")
		if err == nil {
			t.Fatal("Expected error from batch function")
		}

		var keyErr *KeyError
		if !errors.As(err, &keyErr) {
			t.Errorf("Expected KeyError, got %T", err)
		}

		if keyErr.Key != "test" {
			t.Errorf("Expected key 'test', got %s", keyErr.Key)
		}
	})
}
