package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func main() {
	// Example 1: Basic promise usage
	fmt.Println("=== Basic Promise Example ===")
	basicPromiseExample()

	// Example 2: Multiple concurrent promises
	fmt.Println("\n=== Concurrent Promises Example ===")
	concurrentPromisesExample()

	// Example 3: Error handling
	fmt.Println("\n=== Error Handling Example ===")
	errorHandlingExample()

	// Example 4: Deferred promises
	fmt.Println("\n=== Deferred Promises Example ===")
	deferredPromiseExample()

	// Example 5: HTTP client with promises
	fmt.Println("\n=== HTTP Client Example ===")
	httpClientExample()
}

func basicPromiseExample() {
	// Create a promise that resolves after some work
	p := promise.New(func() (string, error) {
		time.Sleep(1 * time.Second)
		return "Hello, World!", nil
	})

	// Do other work while promise is executing
	fmt.Println("Promise created, doing other work...")
	time.Sleep(500 * time.Millisecond)

	// Wait for the promise to resolve
	result, err := p.Await()
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Promise result: %s\n", result)
}

func concurrentPromisesExample() {
	// Create multiple promises
	promises := make([]*promise.Promise[int], 5)

	for i := 0; i < 5; i++ {
		id := i
		promises[i] = promise.New(func() (int, error) {
			// Simulate work with random delay
			delay := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(delay)
			return id * 2, nil
		})
	}

	// Wait for all promises to complete
	fmt.Println("Waiting for all promises to complete...")
	for i, p := range promises {
		result, err := p.Await()
		if err != nil {
			log.Printf("Promise %d error: %v", i, err)
			continue
		}
		fmt.Printf("Promise %d result: %d\n", i, result)
	}
}

func errorHandlingExample() {
	// Create a promise that might fail
	p := promise.New(func() (string, error) {
		if rand.Float32() < 0.5 {
			return "", fmt.Errorf("random failure occurred")
		}
		return "Success!", nil
	})

	result, err := p.Await()
	if err != nil {
		fmt.Printf("Promise failed: %v\n", err)
	} else {
		fmt.Printf("Promise succeeded: %s\n", result)
	}
}

func deferredPromiseExample() {
	// Create a deferred promise
	p := promise.Deferred[string]()

	// Resolve it after some delay in another goroutine
	go func() {
		time.Sleep(1 * time.Second)
		p.Resolve("Deferred result")
	}()

	fmt.Println("Waiting for deferred promise...")
	result, err := p.Await()
	if err != nil {
		log.Printf("Deferred promise error: %v", err)
		return
	}

	fmt.Printf("Deferred promise result: %s\n", result)
}

type APIResponse struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}

func httpClientExample() {
	// Simulate multiple API calls
	urls := []string{
		"https://api.example.com/users/1",
		"https://api.example.com/users/2",
		"https://api.example.com/users/3",
	}

	// Create promises for each API call
	promises := make([]*promise.Promise[APIResponse], len(urls))
	for i, url := range urls {
		id := i + 1
		promises[i] = promise.New(func() (APIResponse, error) {
			// Simulate HTTP request
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

			// Simulate occasional failures
			if rand.Float32() < 0.2 {
				return APIResponse{}, fmt.Errorf("HTTP request failed for %s", url)
			}

			return APIResponse{
				ID:      id,
				Message: fmt.Sprintf("Response from API %d", id),
			}, nil
		})
	}

	// Wait for all API calls to complete
	fmt.Println("Making concurrent API calls...")
	for i, p := range promises {
		response, err := p.Await()
		if err != nil {
			fmt.Printf("API call %d failed: %v\n", i+1, err)
			continue
		}
		fmt.Printf("API call %d response: %+v\n", i+1, response)
	}
}

// DatabaseService demonstrates promise usage with database operations
type DatabaseService struct{}

func (db *DatabaseService) GetUserAsync(ctx context.Context, userID int) *promise.Promise[User] {
	return promise.New(func() (User, error) {
		// Simulate database query
		time.Sleep(100 * time.Millisecond)

		if userID <= 0 {
			return User{}, fmt.Errorf("invalid user ID: %d", userID)
		}

		return User{
			ID:   userID,
			Name: fmt.Sprintf("User %d", userID),
		}, nil
	})
}

func (db *DatabaseService) GetUsersAsync(ctx context.Context, userIDs []int) []*promise.Promise[User] {
	promises := make([]*promise.Promise[User], len(userIDs))
	for i, userID := range userIDs {
		promises[i] = db.GetUserAsync(ctx, userID)
	}
	return promises
}

type User struct {
	ID   int
	Name string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
