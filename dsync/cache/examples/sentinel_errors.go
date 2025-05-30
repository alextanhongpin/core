// Package main demonstrates proper error handling with sentinel errors.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	defer client.Close()

	// Clear any existing data
	client.FlushAll(context.Background())

	ctx := context.Background()
	c := cache.New(client)

	demonstrateSentinelErrors(ctx, c)
}

func demonstrateSentinelErrors(ctx context.Context, c *cache.Cache) {
	fmt.Println("=== Demonstrating Sentinel Error Usage ===")

	key := "test:sentinel"
	value := []byte("test value")

	// 1. Demonstrate ErrNotExist
	fmt.Println("\n1. Testing ErrNotExist:")
	_, err := c.Load(ctx, key)
	if errors.Is(err, cache.ErrNotExist) {
		fmt.Printf("✓ Key '%s' does not exist (expected)\n", key)
	} else {
		fmt.Printf("✗ Unexpected error: %v\n", err)
	}

	// 2. Store a value for further tests
	err = c.Store(ctx, key, value, time.Minute)
	if err != nil {
		log.Fatalf("Failed to store value: %v", err)
	}
	fmt.Printf("✓ Stored value for key '%s'\n", key)

	// 3. Demonstrate ErrExists with StoreOnce
	fmt.Println("\n2. Testing ErrExists:")
	err = c.StoreOnce(ctx, key, []byte("new value"), time.Minute)
	if errors.Is(err, cache.ErrExists) {
		fmt.Printf("✓ Key '%s' already exists (expected)\n", key)
	} else {
		fmt.Printf("✗ Unexpected error: %v\n", err)
	}

	// 4. Demonstrate ErrValueMismatch with CompareAndDelete
	fmt.Println("\n3. Testing ErrValueMismatch:")
	wrongValue := []byte("wrong value")
	err = c.CompareAndDelete(ctx, key, wrongValue)
	if errors.Is(err, cache.ErrValueMismatch) {
		fmt.Printf("✓ CompareAndDelete failed with value mismatch (expected - ERR prefixed)\n")
	} else {
		fmt.Printf("✗ Unexpected error: %v\n", err)
	}

	// 5. Demonstrate successful CompareAndDelete
	fmt.Println("\n4. Testing successful CompareAndDelete:")
	err = c.CompareAndDelete(ctx, key, value)
	if err == nil {
		fmt.Printf("✓ CompareAndDelete succeeded\n")
	} else {
		fmt.Printf("✗ Unexpected error: %v\n", err)
	}

	// 6. Demonstrate ErrNotExist with CompareAndSwap on deleted key
	fmt.Println("\n5. Testing ErrNotExist with CompareAndSwap:")
	err = c.CompareAndSwap(ctx, key, value, []byte("new value"), time.Minute)
	if errors.Is(err, cache.ErrNotExist) {
		fmt.Printf("✓ CompareAndSwap failed because key does not exist (expected - ERR prefixed)\n")
	} else {
		fmt.Printf("✗ Unexpected error: %v\n", err)
	}

	// 7. Demonstrate error handling best practices
	fmt.Println("\n6. Error handling best practices:")
	demonstrateErrorHandling(ctx, c)

	fmt.Println("\n=== Sentinel Error Demo Complete ===")
}

func demonstrateErrorHandling(ctx context.Context, c *cache.Cache) {
	key := "demo:error-handling"

	// Best practice: Use errors.Is() for sentinel error checking
	_, err := c.Load(ctx, key)
	switch {
	case errors.Is(err, cache.ErrNotExist):
		fmt.Println("✓ Handling ErrNotExist with errors.Is()")
		// Handle cache miss - maybe load from database
	case err != nil:
		fmt.Printf("✗ Other error occurred: %v\n", err)
		// Handle other errors
	default:
		fmt.Println("✓ Value loaded successfully")
	}

	// Store a value for CompareAndSwap demo
	testValue := []byte("original")
	c.Store(ctx, key, testValue, time.Minute)

	// Demonstrate multiple error types in one operation
	wrongValue := []byte("wrong")
	err = c.CompareAndSwap(ctx, key, wrongValue, []byte("new"), time.Minute)
	switch {
	case errors.Is(err, cache.ErrNotExist):
		fmt.Println("Key does not exist")
	case errors.Is(err, cache.ErrValueMismatch):
		fmt.Println("✓ Value mismatch detected with sentinel error")
	case errors.Is(err, cache.ErrUnexpectedType):
		fmt.Println("Unexpected data type from Redis")
	case err != nil:
		fmt.Printf("Other error: %v\n", err)
	default:
		fmt.Println("CompareAndSwap succeeded")
	}

	// Clean up
	c.Delete(ctx, key)
}
