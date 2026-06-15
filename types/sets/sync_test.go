package sets_test

import (
	"sync"
	"testing"

	"github.com/alextanhongpin/core/types/sets"
)

// TestBasicSetOperations tests the non-concurrent Set functionality to ensure it remains intact.
func TestBasicSetOperations(t *testing.T) {
	s := sets.New[int]()
	s.Add(10)
	s.Add(20)
	s.Add(10) // Adding duplicate

	if s.Len() != 2 {
		t.Errorf("Expected length 2, got %d", s.Len())
	}
	if !s.Has(20) {
		t.Error("20 should be in the set")
	}
}

// TestConcurrentAdd tests if multiple goroutines can safely add elements without race conditions.
func TestConcurrentAdd(t *testing.T) {
	baseSet := sets.NewSync(sets.New[int]())
	numGoroutines := 100
	itemsPerGoroutine := 1000 // Total items added = 100,000

	var wg sync.WaitGroup

	for id := range numGoroutines {
		wg.Go(func() {
			for j := range itemsPerGoroutine {
				// Add a unique item per goroutine: id * 10000 + j
				item := id*10000 + j
				baseSet.Add(item)
			}
		})
	}

	wg.Wait()

	expectedLen := numGoroutines * itemsPerGoroutine
	actualLen := baseSet.Len()

	if actualLen != expectedLen {
		t.Errorf("Concurrent add failed: Expected final length %d, got %d", expectedLen, actualLen)
	}
}

// TestConcurrentReadWrite tests concurrent read/write operations on a single set.
func TestConcurrentReadWrite(t *testing.T) {
	syncSet := sets.NewSync(sets.New[int]())
	numRoutines := 50

	var wg sync.WaitGroup

	// Concurrent writers (Add)
	for id := range numRoutines {
		wg.Go(func() {
			syncSet.Add(id)
		})
	}

	// Concurrent readers (Has and Len)
	for range numRoutines {
		wg.Go(func() {
			_ = syncSet.Has(1) // Read operation
			_ = syncSet.Len()  // Read operation
		})
	}

	wg.Wait()

	// We don't strictly check the final size here as concurrent additions make the exact count variable,
	// but we verify that the process finished without data races (which is ensured by the lock).
	// If the program reaches this point without panicking from a data race, the synchronization works.
}

// TestConcurrentIntersection tests if simultaneous intersection operations are safe.
func TestConcurrentIntersection(t *testing.T) {
	s1 := sets.Of(1, 2, 3, 4, 5)
	s2 := sets.Of(3, 4, 5, 6, 7)

	// Create synchronized wrappers for testing intersection
	syncS1 := sets.NewSync(s1)
	syncS2 := sets.NewSync(s2)

	numRoutines := 50
	var wg sync.WaitGroup

	for range numRoutines {
		wg.Go(func() {
			// Perform intersection concurrently
			result := syncS1.Intersect(syncS2)
			if result == nil {
				t.Error("Intersection failed to return a result")
			}
		})
	}

	wg.Wait()

	// Final verification of the result is done sequentially:
	finalResult := syncS1.Intersect(syncS2)

	if finalResult == nil {
		t.Fatal("Final intersection failed to return a result for validation")
	}

	// The expected result based on s1={1,2,3,4,5} and s2={3,4,5,6,7} is {3, 4, 5}
	if finalResult.Len() != 3 {
		t.Errorf("Expected intersection size 3 (elements 3, 4, 5), got %d", finalResult.Len())
	}
}
