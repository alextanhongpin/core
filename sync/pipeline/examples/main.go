package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	fmt.Println("=== Pipeline Example ===")

	// Example 1: Basic pipeline with transformation
	fmt.Println("\n1. Basic Pipeline:")
	numbers := pipeline.Range(ctx, 1, 11)
	doubled := pipeline.Transform(numbers, func(n int) int { return n * 2 })
	results := pipeline.ToSlice(doubled)
	fmt.Printf("Numbers doubled: %v\n", results)

	// Example 2: Pipeline with filtering
	fmt.Println("\n2. Pipeline with Filtering:")
	numbers2 := pipeline.Range(ctx, 1, 21)
	evens := pipeline.Filter(numbers2, func(n int) bool { return n%2 == 0 })
	evenResults := pipeline.ToSlice(evens)
	fmt.Printf("Even numbers: %v\n", evenResults)

	// Example 3: Pipeline with batching
	fmt.Println("\n3. Pipeline with Batching:")
	numbers3 := pipeline.Range(ctx, 1, 16)
	batches := pipeline.Batch(5, 500*time.Millisecond, numbers3)

	batchCount := 0
	for batch := range batches {
		batchCount++
		fmt.Printf("Batch %d: %v\n", batchCount, batch)
	}

	// Example 4: Pipeline with error handling
	fmt.Println("\n4. Pipeline with Error Handling:")
	numbers4 := pipeline.Range(ctx, 1, 6)
	withErrors := pipeline.Transform(numbers4, func(n int) pipeline.Result[string] {
		if n == 3 {
			return pipeline.MakeErrorResult[string](fmt.Errorf("error at %d", n))
		}
		return pipeline.MakeSuccessResult(fmt.Sprintf("item-%d", n))
	})

	successful := pipeline.FilterErrors(withErrors, func(err error) {
		fmt.Printf("Error handled: %v\n", err)
	})

	successResults := pipeline.ToSlice(successful)
	fmt.Printf("Successful items: %v\n", successResults)

	// Example 5: Pipeline with parallel processing
	fmt.Println("\n5. Pipeline with Parallel Processing:")
	numbers5 := pipeline.Range(ctx, 1, 6)
	processed := pipeline.Pool(2, numbers5, func(n int) string {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("processed-%d", n)
	})

	processedResults := pipeline.ToSlice(processed)
	fmt.Printf("Processed items: %v\n", processedResults)

	fmt.Println("\n=== Pipeline Example Complete ===")
}
