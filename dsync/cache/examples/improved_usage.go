// Package main demonstrates the improved cache implementation with better error handling and safety.
package main

import (
	"context"
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

	// Demonstrate basic cache operations
	demonstrateBasicOperations(ctx, client)

	// Demonstrate JSON cache operations
	demonstrateJSONOperations(ctx, client)

	// Demonstrate new convenience methods
	demonstrateConvenienceMethods(ctx, client)
}

func demonstrateBasicOperations(ctx context.Context, client *redis.Client) {
	fmt.Println("=== Basic Cache Operations ===")

	c := cache.New(client)

	// Store a value
	key := "user:123"
	value := []byte(`{"id": 123, "name": "John Doe"}`)

	err := c.Store(ctx, key, value, time.Minute)
	if err != nil {
		log.Printf("Store error: %v", err)
		return
	}
	fmt.Printf("Stored value for key: %s\n", key)

	// Load the value
	loaded, err := c.Load(ctx, key)
	if err != nil {
		log.Printf("Load error: %v", err)
		return
	}
	fmt.Printf("Loaded value: %s\n", loaded)

	// Demonstrate LoadOrStore (should load existing)
	newValue := []byte(`{"id": 123, "name": "Jane Doe"}`)
	current, wasLoaded, err := c.LoadOrStore(ctx, key, newValue, time.Minute)
	if err != nil {
		log.Printf("LoadOrStore error: %v", err)
		return
	}
	fmt.Printf("LoadOrStore result - loaded: %t, value: %s\n", wasLoaded, current)

	// Demonstrate CompareAndSwap
	oldValue := []byte(`{"id": 123, "name": "John Doe"}`)
	err = c.CompareAndSwap(ctx, key, oldValue, newValue, time.Minute)
	if err != nil {
		fmt.Printf("CompareAndSwap failed as expected: %v\n", err)
	}

	fmt.Println()
}

func demonstrateJSONOperations(ctx context.Context, client *redis.Client) {
	fmt.Println("=== JSON Cache Operations ===")

	type User struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	jsonCache := cache.NewJSON(client)

	// Store a JSON object
	user := User{ID: 456, Name: "Alice Smith"}
	key := "user:456"

	err := jsonCache.Store(ctx, key, user, time.Minute)
	if err != nil {
		log.Printf("JSON Store error: %v", err)
		return
	}
	fmt.Printf("Stored JSON object for key: %s\n", key)

	// Load the JSON object
	var loadedUser User
	err = jsonCache.Load(ctx, key, &loadedUser)
	if err != nil {
		log.Printf("JSON Load error: %v", err)
		return
	}
	fmt.Printf("Loaded JSON object: %+v\n", loadedUser)

	// Demonstrate JSON LoadOrStore with getter function
	key2 := "user:789"
	var user2 User
	loaded, err := jsonCache.LoadOrStore(ctx, key2, &user2, func() (any, error) {
		fmt.Println("Getter function called - simulating database fetch")
		return User{ID: 789, Name: "Bob Johnson"}, nil
	}, time.Minute)

	if err != nil {
		log.Printf("JSON LoadOrStore error: %v", err)
		return
	}
	fmt.Printf("JSON LoadOrStore result - loaded: %t, user: %+v\n", loaded, user2)

	fmt.Println()
}

func demonstrateConvenienceMethods(ctx context.Context, client *redis.Client) {
	fmt.Println("=== Convenience Methods ===")

	c := cache.New(client)

	key := "test:convenience"
	value := []byte("test value")

	// Store a value
	err := c.Store(ctx, key, value, time.Minute)
	if err != nil {
		log.Printf("Store error: %v", err)
		return
	}

	// Check if key exists
	exists, err := c.Exists(ctx, key)
	if err != nil {
		log.Printf("Exists error: %v", err)
		return
	}
	fmt.Printf("Key %s exists: %t\n", key, exists)

	// Get TTL
	ttl, err := c.TTL(ctx, key)
	if err != nil {
		log.Printf("TTL error: %v", err)
		return
	}
	fmt.Printf("TTL for key %s: %v\n", key, ttl)

	// Delete the key
	deleted, err := c.Delete(ctx, key)
	if err != nil {
		log.Printf("Delete error: %v", err)
		return
	}
	fmt.Printf("Deleted %d key(s)\n", deleted)

	// Verify deletion
	exists, err = c.Exists(ctx, key)
	if err != nil {
		log.Printf("Exists error: %v", err)
		return
	}
	fmt.Printf("Key %s exists after deletion: %t\n", key, exists)

	fmt.Println()
}
