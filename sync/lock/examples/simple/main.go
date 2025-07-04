package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

func main() {
	// Create a new lock manager
	l := lock.New()
	defer l.Stop()

	// Example 1: Basic named locking
	fmt.Println("=== Basic Named Locking ===")

	var wg sync.WaitGroup
	counter := 0

	// Multiple goroutines working on the same resource
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Get a lock for the shared resource
			locker := l.Get("shared-counter")
			locker.Lock()
			defer locker.Unlock()

			// Simulate some work
			oldValue := counter
			time.Sleep(10 * time.Millisecond)
			counter = oldValue + 1

			fmt.Printf("Goroutine %d: counter = %d\n", id, counter)
		}(i)
	}

	wg.Wait()
	fmt.Printf("Final counter value: %d\n", counter)

	// Example 2: Multiple named locks
	fmt.Println("\n=== Multiple Named Locks ===")

	resources := []string{"resource-A", "resource-B", "resource-C"}

	for _, resource := range resources {
		wg.Add(1)
		go func(res string) {
			defer wg.Done()

			// Each resource has its own lock
			locker := l.Get(res)
			locker.Lock()
			defer locker.Unlock()

			fmt.Printf("Working with %s\n", res)
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("Done with %s\n", res)
		}(resource)
	}

	wg.Wait()

	// Example 3: Lock metrics
	fmt.Println("\n=== Lock Metrics ===")
	metrics := l.Metrics()
	fmt.Printf("Active locks: %d\n", metrics.ActiveLocks)
	fmt.Printf("Total locks created: %d\n", metrics.TotalLocks)
	fmt.Printf("Lock acquisitions: %d\n", metrics.LockAcquisitions)
	fmt.Printf("Average wait time: %v\n", metrics.AverageWaitTime)

	fmt.Println("\n=== Example Complete ===")
}
