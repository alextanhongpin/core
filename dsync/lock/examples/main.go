// Package main demonstrates various lock usage patterns
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Setup Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Clean up for demo
	client.FlushAll(context.Background())

	fmt.Println("=== Distributed Lock Examples ===\n")

	// 1. Basic locking
	demonstrateBasicLocking(client)

	// 2. Lock with waiting
	demonstrateLockWithWaiting(client)

	// 3. PubSub optimization
	demonstratePubSubOptimization(client)

	// 4. Error handling
	demonstrateErrorHandling(client)

	// 5. Manual lock control
	demonstrateManualLockControl(client)

	// 6. Concurrent access simulation
	demonstrateConcurrentAccess(client)

	fmt.Println("\n=== All Examples Complete ===")
}

func demonstrateBasicLocking(client *redis.Client) {
	fmt.Println("1. Basic Locking Example")
	locker := lock.New(client)

	ctx := context.Background()
	key := "basic-lock-example"

	err := locker.Do(ctx, key, func(ctx context.Context) error {
		fmt.Println("   Executing critical section...")
		time.Sleep(1 * time.Second)
		fmt.Println("   Critical section completed")
		return nil
	}, &lock.LockOption{
		Lock:         5 * time.Second,
		Wait:         0, // Don't wait if busy
		RefreshRatio: 0.8,
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Println("   Success!")
	}
	fmt.Println()
}

func demonstrateLockWithWaiting(client *redis.Client) {
	fmt.Println("2. Lock with Waiting Example")
	locker := lock.New(client)

	ctx := context.Background()
	key := "waiting-lock-example"

	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine - holds lock for 2 seconds
	go func() {
		defer wg.Done()
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			fmt.Println("   Worker 1: Acquired lock, working...")
			time.Sleep(2 * time.Second)
			fmt.Println("   Worker 1: Completed work")
			return nil
		}, &lock.LockOption{
			Lock:         10 * time.Second,
			Wait:         0,
			RefreshRatio: 0.8,
		})
		if err != nil {
			fmt.Printf("   Worker 1 Error: %v\n", err)
		}
	}()

	// Second goroutine - waits for lock
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond) // Start after first worker

		err := locker.Do(ctx, key, func(ctx context.Context) error {
			fmt.Println("   Worker 2: Acquired lock, working...")
			time.Sleep(1 * time.Second)
			fmt.Println("   Worker 2: Completed work")
			return nil
		}, &lock.LockOption{
			Lock:         10 * time.Second,
			Wait:         5 * time.Second, // Wait up to 5 seconds
			RefreshRatio: 0.8,
		})
		if err != nil {
			fmt.Printf("   Worker 2 Error: %v\n", err)
		}
	}()

	wg.Wait()
	fmt.Println("   Both workers completed")
	fmt.Println()
}

func demonstratePubSubOptimization(client *redis.Client) {
	fmt.Println("3. PubSub Optimization Example")
	pubsubLocker := lock.NewPubSub(client)

	ctx := context.Background()
	key := "pubsub-lock-example"

	var wg sync.WaitGroup
	wg.Add(3)

	// Simulate multiple workers competing for the same lock
	for i := 1; i <= 3; i++ {
		go func(workerID int) {
			defer wg.Done()

			start := time.Now()
			err := pubsubLocker.Do(ctx, key, func(ctx context.Context) error {
				fmt.Printf("   Worker %d: Acquired lock after %v\n", workerID, time.Since(start))
				time.Sleep(500 * time.Millisecond)
				return nil
			}, &lock.LockOption{
				Lock:         5 * time.Second,
				Wait:         3 * time.Second,
				RefreshRatio: 0.8,
			})
			if err != nil {
				fmt.Printf("   Worker %d Error: %v\n", workerID, err)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("   PubSub optimization completed")
	fmt.Println()
}

func demonstrateErrorHandling(client *redis.Client) {
	fmt.Println("4. Error Handling Example")
	locker := lock.New(client)

	ctx := context.Background()
	key := "error-handling-example"

	// First acquire a lock
	go func() {
		locker.Do(ctx, key, func(ctx context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		}, &lock.LockOption{
			Lock: 10 * time.Second,
			Wait: 0,
		})
	}()

	time.Sleep(100 * time.Millisecond) // Ensure first lock is acquired

	// Try to acquire the same lock without waiting
	err := locker.Do(ctx, key, func(ctx context.Context) error {
		return nil
	}, &lock.LockOption{
		Lock: 5 * time.Second,
		Wait: 0,
	})

	fmt.Printf("   Handling different error types:\n")
	switch {
	case errors.Is(err, lock.ErrLocked):
		fmt.Printf("   - Resource is busy (ErrLocked): %v\n", err)
	case errors.Is(err, lock.ErrLockWaitTimeout):
		fmt.Printf("   - Timeout waiting (ErrLockWaitTimeout): %v\n", err)
	case errors.Is(err, lock.ErrLockTimeout):
		fmt.Printf("   - Lock expired (ErrLockTimeout): %v\n", err)
	default:
		fmt.Printf("   - Other error: %v\n", err)
	}
	fmt.Println()
}

func demonstrateManualLockControl(client *redis.Client) {
	fmt.Println("5. Manual Lock Control Example")
	locker := lock.New(client)

	ctx := context.Background()
	key := "manual-lock-example"
	token := "my-unique-token-123"

	// Try to acquire lock
	fmt.Println("   Attempting to acquire lock...")
	err := locker.TryLock(ctx, key, token, 10*time.Second)
	if err != nil {
		fmt.Printf("   Failed to acquire lock: %v\n", err)
		return
	}
	fmt.Println("   Lock acquired successfully!")

	// Extend lock
	fmt.Println("   Extending lock...")
	err = locker.Extend(ctx, key, token, 10*time.Second)
	if err != nil {
		fmt.Printf("   Failed to extend lock: %v\n", err)
	} else {
		fmt.Println("   Lock extended successfully!")
	}

	// Simulate work
	time.Sleep(1 * time.Second)

	// Release lock
	fmt.Println("   Releasing lock...")
	err = locker.Unlock(ctx, key, token)
	if err != nil {
		fmt.Printf("   Failed to unlock: %v\n", err)
	} else {
		fmt.Println("   Lock released successfully!")
	}
	fmt.Println()
}

func demonstrateConcurrentAccess(client *redis.Client) {
	fmt.Println("6. Concurrent Access Simulation")
	locker := lock.New(client)

	ctx := context.Background()
	key := "concurrent-access-example"

	// Shared counter
	counter := 0
	var mu sync.Mutex

	// Function to increment counter safely
	incrementCounter := func() {
		mu.Lock()
		counter++
		mu.Unlock()
	}

	var wg sync.WaitGroup
	numWorkers := 5

	fmt.Printf("   Starting %d workers...\n", numWorkers)
	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			err := locker.Do(ctx, key, func(ctx context.Context) error {
				fmt.Printf("   Worker %d: Processing...\n", workerID)

				// Simulate some work
				time.Sleep(200 * time.Millisecond)

				// Increment counter
				incrementCounter()

				return nil
			}, &lock.LockOption{
				Lock:         5 * time.Second,
				Wait:         10 * time.Second,
				RefreshRatio: 0.8,
			})

			if err != nil {
				fmt.Printf("   Worker %d Error: %v\n", workerID, err)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("   All workers completed in %v\n", elapsed)
	fmt.Printf("   Final counter value: %d\n", counter)
	fmt.Println()
}
