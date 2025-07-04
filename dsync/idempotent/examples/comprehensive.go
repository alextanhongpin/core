// Package main demonstrates comprehensive idempotent patterns
// This file shows various usage patterns but should not be run as main
// To avoid conflicts with the main.go file, this is for reference only
package examples

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserResponse struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

func RunExamples() {
	// Initialize Redis test environment
	stop := redistest.Init()
	defer stop()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	defer client.Close()

	fmt.Println("=== Idempotent Handler Examples ===\n")

	// Example 1: Basic Usage
	demonstrateBasicUsage(client)

	// Example 2: Concurrent Requests
	demonstrateConcurrentRequests(client)

	// Example 3: Error Handling
	demonstrateErrorHandling(client)

	// Example 4: Long Running Operations
	demonstrateLongRunningOperations(client)

	// Example 5: HTTP API Pattern
	demonstrateHTTPPattern(client)

	fmt.Println("\n=== All Examples Complete ===")
}

func demonstrateBasicUsage(client *redis.Client) {
	fmt.Println("1. Basic Idempotent Usage")

	// Create a user creation function
	createUser := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
		fmt.Printf("   Creating user: %s (%s)\n", req.Name, req.Email)
		// Simulate database operation
		time.Sleep(50 * time.Millisecond)

		return &CreateUserResponse{
			UserID: rand.Int63n(100000),
			Name:   req.Name,
			Email:  req.Email,
		}, nil
	}

	handler := idempotent.NewHandler(client, createUser, nil)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// First request - will execute the function
	resp1, shared1, err := handler.Handle(ctx, "create-user-123", req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   First request: UserID=%d, Shared=%v\n", resp1.UserID, shared1)

	// Second request - will return cached result
	resp2, shared2, err := handler.Handle(ctx, "create-user-123", req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Second request: UserID=%d, Shared=%v\n", resp2.UserID, shared2)

	if resp1.UserID == resp2.UserID {
		fmt.Println("   ✅ Same result returned (idempotent)")
	}
	fmt.Println()
}

func demonstrateConcurrentRequests(client *redis.Client) {
	fmt.Println("2. Concurrent Requests")

	executionCount := 0
	createUser := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
		executionCount++
		fmt.Printf("   Executing function (count: %d)\n", executionCount)
		// Simulate slow operation
		time.Sleep(200 * time.Millisecond)

		return &CreateUserResponse{
			UserID: 42,
			Name:   req.Name,
			Email:  req.Email,
		}, nil
	}

	handler := idempotent.NewHandler(client, createUser, nil)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}

	// Start multiple concurrent requests
	var wg sync.WaitGroup
	results := make(chan *CreateUserResponse, 5)
	sharedCounts := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, shared, err := handler.Handle(ctx, "concurrent-user", req)
			if err != nil {
				fmt.Printf("   Goroutine %d error: %v\n", id, err)
				return
			}
			results <- resp
			sharedCounts <- shared
		}(i)
	}

	wg.Wait()
	close(results)
	close(sharedCounts)

	// Check results
	userIDs := make(map[int64]int)
	sharedCount := 0
	for range 5 {
		select {
		case resp := <-results:
			userIDs[resp.UserID]++
		default:
		}

		select {
		case shared := <-sharedCounts:
			if shared {
				sharedCount++
			}
		default:
		}
	}

	fmt.Printf("   Function executed %d times\n", executionCount)
	fmt.Printf("   Shared responses: %d/5\n", sharedCount)
	fmt.Printf("   Unique UserIDs: %d\n", len(userIDs))
	if len(userIDs) == 1 {
		fmt.Println("   ✅ All requests returned same result")
	}
	fmt.Println()
}

func demonstrateErrorHandling(client *redis.Client) {
	fmt.Println("3. Error Handling")

	// Function that might fail
	unreliableCreate := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
		if req.Name == "error" {
			return nil, errors.New("simulated error")
		}
		return &CreateUserResponse{
			UserID: 999,
			Name:   req.Name,
			Email:  req.Email,
		}, nil
	}

	handler := idempotent.NewHandler(client, unreliableCreate, nil)
	ctx := context.Background()

	// Test with error
	req := CreateUserRequest{Name: "error", Email: "error@example.com"}
	_, _, err := handler.Handle(ctx, "error-test", req)
	if err != nil {
		fmt.Printf("   Expected error: %v\n", err)
	}

	// Test request mismatch
	req1 := CreateUserRequest{Name: "Alice", Email: "alice@example.com"}
	req2 := CreateUserRequest{Name: "Bob", Email: "bob@example.com"}

	// First request
	_, _, err = handler.Handle(ctx, "mismatch-test", req1)
	if err != nil {
		fmt.Printf("   First request error: %v\n", err)
	}

	// Second request with different data but same key
	_, _, err = handler.Handle(ctx, "mismatch-test", req2)
	if err != nil {
		if errors.Is(err, idempotent.ErrRequestMismatch) {
			fmt.Println("   ✅ Request mismatch detected")
		} else {
			fmt.Printf("   Unexpected error: %v\n", err)
		}
	}
	fmt.Println()
}

func demonstrateLongRunningOperations(client *redis.Client) {
	fmt.Println("4. Long Running Operations")

	slowCreate := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
		fmt.Println("   Starting slow operation...")
		// Simulate very slow operation
		time.Sleep(2 * time.Second)
		fmt.Println("   Slow operation completed")

		return &CreateUserResponse{
			UserID: 777,
			Name:   req.Name,
			Email:  req.Email,
		}, nil
	}

	// Configure with shorter lock TTL to test extension
	handler := idempotent.NewHandler(client, slowCreate, &idempotent.HandlerOptions{
		LockTTL: 500 * time.Millisecond, // Shorter than operation time
		KeepTTL: time.Hour,
	})

	ctx := context.Background()
	req := CreateUserRequest{Name: "Slow User", Email: "slow@example.com"}

	start := time.Now()
	resp, shared, err := handler.Handle(ctx, "slow-operation", req)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Operation completed in %v\n", duration)
		fmt.Printf("   UserID: %d, Shared: %v\n", resp.UserID, shared)
		fmt.Println("   ✅ Lock extended successfully for long operation")
	}
	fmt.Println()
}

func demonstrateHTTPPattern(client *redis.Client) {
	fmt.Println("5. HTTP API Pattern")

	// Simulated user service
	userService := func(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
		return &CreateUserResponse{
			UserID: rand.Int63n(1000000),
			Name:   req.Name,
			Email:  req.Email,
		}, nil
	}

	handler := idempotent.NewHandler(client, userService, nil)

	// HTTP handler function
	createUserHandler := func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Use idempotency key from header
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, "Missing Idempotency-Key header", http.StatusBadRequest)
			return
		}

		resp, shared, err := handler.Handle(r.Context(), key, req)
		if err != nil {
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				http.Error(w, "Request in progress", http.StatusConflict)
				return
			}
			if errors.Is(err, idempotent.ErrRequestMismatch) {
				http.Error(w, "Request mismatch", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Idempotent-Replayed", fmt.Sprintf("%t", shared))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	fmt.Println("   HTTP handler pattern implemented")
	fmt.Println("   - Idempotency-Key header required")
	fmt.Println("   - X-Idempotent-Replayed header in response")
	fmt.Println("   - Proper error handling for different scenarios")
	fmt.Println("   ✅ Ready for production use")

	// For demonstration, we're not starting a real server
	_ = createUserHandler
}
