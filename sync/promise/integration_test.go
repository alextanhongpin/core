package promise_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

// TestIntegrationScenario tests a comprehensive real-world scenario
func TestIntegrationScenario(t *testing.T) {
	// Simulate a service that fetches user data from multiple sources
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create promises for different data sources
	userPromise := promise.NewWithContext(ctx, func(ctx context.Context) (User, error) {
		time.Sleep(50 * time.Millisecond)
		return User{ID: 1, Name: "John Doe"}, nil
	})

	profilePromise := promise.NewWithContext(ctx, func(ctx context.Context) (Profile, error) {
		time.Sleep(75 * time.Millisecond)
		return Profile{UserID: 1, Bio: "Software Engineer"}, nil
	})

	settingsPromise := promise.NewWithContext(ctx, func(ctx context.Context) (Settings, error) {
		time.Sleep(25 * time.Millisecond)
		return Settings{UserID: 1, Theme: "dark"}, nil
	})

	// Use promise collection to wait for all data
	allPromises := []*promise.Promise[any]{
		promise.NewWithContext(ctx, func(ctx context.Context) (any, error) {
			return userPromise.Await()
		}),
		promise.NewWithContext(ctx, func(ctx context.Context) (any, error) {
			return profilePromise.Await()
		}),
		promise.NewWithContext(ctx, func(ctx context.Context) (any, error) {
			return settingsPromise.Await()
		}),
	}

	// Wait for all data to be fetched
	allResults := promise.Promises[any](allPromises).AllSettledWithContext(ctx)

	// Verify all results are successful
	if len(allResults) != 3 {
		t.Fatalf("expected 3 results, got %d", len(allResults))
	}

	for i, result := range allResults {
		if result.IsRejected() {
			t.Fatalf("result %d was rejected: %v", i, result.Err)
		}
	}

	// Test pool functionality
	pool := promise.NewPoolWithContext[int](ctx, 2)

	// Submit multiple tasks
	for i := 0; i < 5; i++ {
		val := i
		err := pool.DoWithContext(ctx, func(ctx context.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return val * 2, nil
		})
		if err != nil {
			t.Fatalf("unexpected error submitting task: %v", err)
		}
	}

	// Wait for all pool tasks to complete
	poolResults := pool.AllSettled()
	if len(poolResults) != 5 {
		t.Fatalf("expected 5 pool results, got %d", len(poolResults))
	}

	// Test Map functionality
	userMap := promise.NewMap[User]()

	// Store some user promises
	userMap.Store("user1", promise.Resolve(User{ID: 1, Name: "Alice"}))
	userMap.Store("user2", promise.Resolve(User{ID: 2, Name: "Bob"}))

	// Load a user
	userPromise, found := userMap.Load("user1")
	if !found {
		t.Fatal("expected to find user1")
	}
	user, err := userPromise.Await()
	if err != nil {
		t.Fatalf("unexpected error getting user: %v", err)
	}
	if user.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", user.Name)
	}

	// Test Group functionality
	group := promise.NewGroup[string]()

	// Execute a task
	result, err := group.DoWithContext("process_text", ctx, func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "processed", nil
	})
	if err != nil {
		t.Fatalf("unexpected error from group: %v", err)
	}
	if result != "processed" {
		t.Fatalf("expected 'processed', got %s", result)
	}

	// Test error handling with panic recovery
	panicPromise := promise.New(func() (string, error) {
		panic("simulated panic")
	})

	_, err = panicPromise.Await()
	if err == nil {
		t.Fatal("expected error from panic")
	}
	// Check if it's a panic error by checking the error message
	if !isRecoveryError(err) {
		t.Fatalf("expected panic recovery error, got %v", err)
	}

	// Test timeout scenario
	timeoutPromise := promise.New(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "timeout test", nil
	})

	_, err = timeoutPromise.AwaitWithTimeout(10 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Check if it's a timeout error (either context.DeadlineExceeded or custom timeout error)
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() != "promise: timeout" {
		t.Fatalf("expected timeout error, got %v", err)
	}

	t.Log("Integration test passed successfully!")
}

// Helper function to check if error is from panic recovery
func isRecoveryError(err error) bool {
	return err != nil && (err.Error() == "panic recovered: simulated panic" ||
		err.Error() == "panic recovered: simulated panic\n" ||
		err.Error() == "promise: panic occurred")
}

// Test data structures
type User struct {
	ID   int
	Name string
}

type Profile struct {
	UserID int
	Bio    string
}

type Settings struct {
	UserID int
	Theme  string
}
