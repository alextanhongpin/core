package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/singleflight"
)

func main() {
	// Example 1: Basic singleflight usage
	fmt.Println("=== Basic Singleflight Example ===")
	basicSingleflightExample()

	// Example 2: Cache with singleflight
	fmt.Println("\n=== Cache with Singleflight Example ===")
	cacheExample()

	// Example 3: Database query deduplication
	fmt.Println("\n=== Database Query Deduplication Example ===")
	databaseExample()

	// Example 4: HTTP request deduplication
	fmt.Println("\n=== HTTP Request Deduplication Example ===")
	httpExample()
}

func basicSingleflightExample() {
	ctx := context.Background()
	group := singleflight.New[string]()

	// Function that simulates expensive work
	expensiveWork := func(ctx context.Context) (string, error) {
		fmt.Println("Doing expensive work...")
		time.Sleep(2 * time.Second)
		return "result", nil
	}

	// Start multiple concurrent calls with the same key
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			start := time.Now()
			result, shared, err := group.Do(ctx, "key1", expensiveWork)
			if err != nil {
				fmt.Printf("Worker %d: Error: %v\n", id, err)
				return
			}

			fmt.Printf("Worker %d: Result: %s, Shared: %t, Duration: %v\n",
				id, result, shared, time.Since(start))
		}(i)
	}

	wg.Wait()
	fmt.Println("All workers completed")
}

func cacheExample() {
	ctx := context.Background()
	cache := NewCache()

	// Simulate multiple concurrent requests for the same data
	var wg sync.WaitGroup
	keys := []string{"user:1", "user:2", "user:1", "user:2", "user:3"}

	for i, key := range keys {
		wg.Add(1)
		go func(id int, key string) {
			defer wg.Done()

			start := time.Now()
			value, err := cache.Get(ctx, key)
			if err != nil {
				fmt.Printf("Request %d: Error getting %s: %v\n", id, key, err)
				return
			}

			fmt.Printf("Request %d: Got %s = %s in %v\n",
				id, key, value, time.Since(start))
		}(i, key)
	}

	wg.Wait()
	fmt.Println("All cache requests completed")
}

func databaseExample() {
	ctx := context.Background()
	db := NewDatabase()

	// Simulate multiple concurrent database queries
	var wg sync.WaitGroup
	userIDs := []int{1, 2, 1, 1, 3, 2, 1}

	for i, userID := range userIDs {
		wg.Add(1)
		go func(requestID, userID int) {
			defer wg.Done()

			start := time.Now()
			user, err := db.GetUser(ctx, userID)
			if err != nil {
				fmt.Printf("Request %d: Error getting user %d: %v\n",
					requestID, userID, err)
				return
			}

			fmt.Printf("Request %d: Got user %d (%s) in %v\n",
				requestID, user.ID, user.Name, time.Since(start))
		}(i, userID)
	}

	wg.Wait()
	fmt.Println("All database queries completed")
}

func httpExample() {
	ctx := context.Background()
	client := NewHTTPClient()

	// Simulate multiple concurrent HTTP requests
	var wg sync.WaitGroup
	urls := []string{
		"https://api.example.com/users/1",
		"https://api.example.com/users/2",
		"https://api.example.com/users/1", // Duplicate
		"https://api.example.com/users/1", // Duplicate
		"https://api.example.com/users/3",
	}

	for i, url := range urls {
		wg.Add(1)
		go func(requestID int, url string) {
			defer wg.Done()

			start := time.Now()
			response, err := client.Get(ctx, url)
			if err != nil {
				fmt.Printf("Request %d: Error fetching %s: %v\n",
					requestID, url, err)
				return
			}

			fmt.Printf("Request %d: Got %s (%d bytes) in %v\n",
				requestID, url, len(response), time.Since(start))
		}(i, url)
	}

	wg.Wait()
	fmt.Println("All HTTP requests completed")
}

// Cache demonstrates singleflight usage with caching
type Cache struct {
	data  map[string]string
	mu    sync.RWMutex
	group *singleflight.Group[string]
}

func NewCache() *Cache {
	return &Cache{
		data:  make(map[string]string),
		group: singleflight.New[string](),
	}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	// Check cache first
	c.mu.RLock()
	if value, exists := c.data[key]; exists {
		c.mu.RUnlock()
		fmt.Printf("Cache hit for key: %s\n", key)
		return value, nil
	}
	c.mu.RUnlock()

	// Cache miss, use singleflight to load
	value, shared, err := c.group.Do(ctx, key, func(ctx context.Context) (string, error) {
		fmt.Printf("Loading data for key: %s\n", key)

		// Simulate expensive operation
		time.Sleep(1 * time.Second)

		value := fmt.Sprintf("data_for_%s", key)

		// Update cache
		c.mu.Lock()
		c.data[key] = value
		c.mu.Unlock()

		return value, nil
	})

	if err != nil {
		return "", err
	}

	if shared {
		fmt.Printf("Value for key %s shared from concurrent call\n", key)
	}

	return value, nil
}

// Database demonstrates singleflight usage with database queries
type Database struct {
	group *singleflight.Group[User]
}

type User struct {
	ID   int
	Name string
}

func NewDatabase() *Database {
	return &Database{
		group: singleflight.New[User](),
	}
}

func (db *Database) GetUser(ctx context.Context, userID int) (User, error) {
	key := fmt.Sprintf("user:%d", userID)

	user, shared, err := db.group.Do(ctx, key, func(ctx context.Context) (User, error) {
		fmt.Printf("Executing database query for user %d\n", userID)

		// Simulate database query
		time.Sleep(500 * time.Millisecond)

		return User{
			ID:   userID,
			Name: fmt.Sprintf("User %d", userID),
		}, nil
	})

	if err != nil {
		return User{}, err
	}

	if shared {
		fmt.Printf("User %d result shared from concurrent call\n", userID)
	}

	return user, nil
}

// HTTPClient demonstrates singleflight usage with HTTP requests
type HTTPClient struct {
	group *singleflight.Group[string]
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		group: singleflight.New[string](),
	}
}

func (c *HTTPClient) Get(ctx context.Context, url string) (string, error) {
	response, shared, err := c.group.Do(ctx, url, func(ctx context.Context) (string, error) {
		fmt.Printf("Making HTTP request to %s\n", url)

		// Simulate HTTP request
		time.Sleep(800 * time.Millisecond)

		// Simulate response
		response := fmt.Sprintf("Response from %s", url)

		return response, nil
	})

	if err != nil {
		return "", err
	}

	if shared {
		fmt.Printf("Response for %s shared from concurrent call\n", url)
	}

	return response, nil
}
