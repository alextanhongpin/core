package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func TestResult(t *testing.T) {
	// Test successful result
	result := pipeline.MakeSuccessResult(42)
	if result.IsError() {
		t.Error("Expected success result")
	}
	if !result.IsSuccess() {
		t.Error("Expected success result")
	}

	data, err := result.Unwrap()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if data != 42 {
		t.Errorf("Expected 42, got %v", data)
	}

	// Test error result
	testErr := errors.New("test error")
	errorResult := pipeline.MakeErrorResult[int](testErr)
	if !errorResult.IsError() {
		t.Error("Expected error result")
	}
	if errorResult.IsSuccess() {
		t.Error("Expected error result")
	}

	_, err = errorResult.Unwrap()
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestBuffer(t *testing.T) {
	in := make(chan int)
	out := pipeline.Buffer(5, in)

	// Send some values
	go func() {
		defer close(in)
		for i := 0; i < 10; i++ {
			in <- i
		}
	}()

	// Collect results
	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

func TestQueue(t *testing.T) {
	in := make(chan int)
	out := pipeline.Queue(3, in)

	go func() {
		defer close(in)
		for i := 0; i < 5; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}

func TestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	in := make(chan int)
	out := pipeline.WithContext(ctx, in)

	go func() {
		defer close(in)
		for i := 0; i < 10; i++ {
			in <- i
			if i == 4 {
				cancel() // Cancel after sending 5 items
			}
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	// Should get at most 5 items before cancellation
	if len(results) > 5 {
		t.Errorf("Expected at most 5 results, got %d", len(results))
	}
}

func TestWithTimeout(t *testing.T) {
	in := make(chan int)
	out := pipeline.WithTimeout(100*time.Millisecond, in)

	go func() {
		defer close(in)
		in <- 1
		time.Sleep(200 * time.Millisecond) // This should timeout
		in <- 2
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	// Should only get the first item
	if len(results) != 1 || results[0] != 1 {
		t.Errorf("Expected [1], got %v", results)
	}
}

func TestTransform(t *testing.T) {
	in := make(chan int)
	out := pipeline.Transform(in, func(x int) int { return x * 2 })

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	expected := []int{2, 4, 6, 8, 10}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("Expected %v, got %v", expected, results)
		}
	}
}

func TestMap(t *testing.T) {
	in := make(chan int)
	out := pipeline.Map(in, func(x int) string {
		return fmt.Sprintf("item-%d", x)
	})

	go func() {
		defer close(in)
		for i := 1; i <= 3; i++ {
			in <- i
		}
	}()

	var results []string
	for v := range out {
		results = append(results, v)
	}

	expected := []string{"item-1", "item-2", "item-3"}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestFilter(t *testing.T) {
	in := make(chan int)
	out := pipeline.Filter(in, func(x int) bool { return x%2 == 0 })

	go func() {
		defer close(in)
		for i := 1; i <= 10; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	expected := []int{2, 4, 6, 8, 10}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestTake(t *testing.T) {
	in := make(chan int)
	out := pipeline.Take(3, in)

	go func() {
		defer close(in)
		for i := 1; i <= 10; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestSkip(t *testing.T) {
	in := make(chan int)
	out := pipeline.Skip(3, in)

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	expected := []int{4, 5}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestDistinct(t *testing.T) {
	in := make(chan int)
	out := pipeline.Distinct(in)

	go func() {
		defer close(in)
		values := []int{1, 2, 2, 3, 3, 3, 4}
		for _, v := range values {
			in <- v
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	expected := []int{1, 2, 3, 4}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestPool(t *testing.T) {
	in := make(chan int)
	out := pipeline.Pool(3, in, func(x int) int {
		time.Sleep(10 * time.Millisecond)
		return x * 2
	})

	start := time.Now()
	go func() {
		defer close(in)
		for i := 1; i <= 6; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	elapsed := time.Since(start)

	// With 3 workers and 6 items, should take ~20ms (2 batches)
	if elapsed > 50*time.Millisecond {
		t.Errorf("Pool took too long: %v", elapsed)
	}

	if len(results) != 6 {
		t.Errorf("Expected 6 results, got %d", len(results))
	}
}

func TestPoolWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	in := make(chan int)
	out := pipeline.PoolWithContext(ctx, 2, in, func(ctx context.Context, x int) int {
		select {
		case <-ctx.Done():
			return -1
		case <-time.After(10 * time.Millisecond):
			return x * 2
		}
	})

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
			if i == 3 {
				cancel()
			}
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	// Should get some results before cancellation
	if len(results) == 0 {
		t.Error("Expected some results")
	}
}

func TestFanOut(t *testing.T) {
	in := make(chan int)
	outputs := pipeline.FanOut(3, in)

	go func() {
		defer close(in)
		for i := 1; i <= 9; i++ {
			in <- i
		}
	}()

	var wg sync.WaitGroup
	var results [3][]int

	for i, out := range outputs {
		wg.Add(1)
		go func(idx int, ch <-chan int) {
			defer wg.Done()
			for v := range ch {
				results[idx] = append(results[idx], v)
			}
		}(i, out)
	}

	wg.Wait()

	// Each output should get some values
	totalItems := 0
	for i := 0; i < 3; i++ {
		totalItems += len(results[i])
	}

	if totalItems != 9 {
		t.Errorf("Expected 9 total items, got %d", totalItems)
	}
}

func TestFanIn(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	out := pipeline.FanIn(ch1, ch2, ch3)

	go func() {
		defer close(ch1)
		for i := 1; i <= 3; i++ {
			ch1 <- i
		}
	}()

	go func() {
		defer close(ch2)
		for i := 4; i <= 6; i++ {
			ch2 <- i
		}
	}()

	go func() {
		defer close(ch3)
		for i := 7; i <= 9; i++ {
			ch3 <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 9 {
		t.Errorf("Expected 9 results, got %d", len(results))
	}
}

func TestThrottle(t *testing.T) {
	in := make(chan int)
	out := pipeline.Throttle(50*time.Millisecond, in)

	start := time.Now()
	go func() {
		defer close(in)
		for i := 1; i <= 3; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	elapsed := time.Since(start)

	// Should take at least 150ms for 3 items with 50ms throttle
	if elapsed < 100*time.Millisecond {
		t.Errorf("Throttling too fast: %v", elapsed)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestRateLimit(t *testing.T) {
	in := make(chan int)
	out := pipeline.RateLimit(10, in) // 10 items per second

	start := time.Now()
	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	elapsed := time.Since(start)

	// Should take at least 400ms for 5 items at 10/sec
	if elapsed < 300*time.Millisecond {
		t.Errorf("Rate limiting too fast: %v", elapsed)
	}
}

func TestBatch(t *testing.T) {
	in := make(chan int)
	out := pipeline.Batch(3, 100*time.Millisecond, in)

	go func() {
		defer close(in)
		for i := 1; i <= 7; i++ {
			in <- i
			time.Sleep(10 * time.Millisecond)
		}
	}()

	var batches [][]int
	for batch := range out {
		batches = append(batches, batch)
	}

	// Should get multiple batches
	if len(batches) < 2 {
		t.Errorf("Expected at least 2 batches, got %d", len(batches))
	}

	// First batch should be size 3
	if len(batches[0]) != 3 {
		t.Errorf("Expected first batch size 3, got %d", len(batches[0]))
	}
}

func TestBatchDistinct(t *testing.T) {
	in := make(chan int)
	out := pipeline.BatchDistinct(3, 100*time.Millisecond, in)

	go func() {
		defer close(in)
		values := []int{1, 1, 2, 2, 3, 3, 4}
		for _, v := range values {
			in <- v
			time.Sleep(10 * time.Millisecond)
		}
	}()

	var batches [][]int
	for batch := range out {
		batches = append(batches, batch)
	}

	// Should get distinct values in batches
	if len(batches) < 1 {
		t.Error("Expected at least 1 batch")
	}

	// First batch should have 3 distinct values
	if len(batches[0]) != 3 {
		t.Errorf("Expected first batch size 3, got %d", len(batches[0]))
	}
}

func TestTee(t *testing.T) {
	in := make(chan int)
	out1, out2 := pipeline.Tee(in)

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	var wg sync.WaitGroup
	var results1, results2 []int

	wg.Add(2)
	go func() {
		defer wg.Done()
		for v := range out1 {
			results1 = append(results1, v)
		}
	}()

	go func() {
		defer wg.Done()
		for v := range out2 {
			results2 = append(results2, v)
		}
	}()

	wg.Wait()

	if len(results1) != 5 || len(results2) != 5 {
		t.Errorf("Expected both outputs to have 5 items, got %d and %d", len(results1), len(results2))
	}

	// Both should have the same values
	for i := 0; i < 5; i++ {
		if results1[i] != results2[i] {
			t.Errorf("Outputs differ at index %d: %d vs %d", i, results1[i], results2[i])
		}
	}
}

func TestFlatMap(t *testing.T) {
	in := make(chan pipeline.Result[int])
	out := pipeline.FlatMap(in)

	go func() {
		defer close(in)
		in <- pipeline.MakeSuccessResult(1)
		in <- pipeline.MakeErrorResult[int](errors.New("error"))
		in <- pipeline.MakeSuccessResult(2)
		in <- pipeline.MakeSuccessResult(3)
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	expected := []int{1, 2, 3}
	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestFilterErrors(t *testing.T) {
	in := make(chan pipeline.Result[int])

	var errorCount int64
	out := pipeline.FilterErrors(in, func(err error) {
		atomic.AddInt64(&errorCount, 1)
	})

	go func() {
		defer close(in)
		in <- pipeline.MakeSuccessResult(1)
		in <- pipeline.MakeErrorResult[int](errors.New("error1"))
		in <- pipeline.MakeSuccessResult(2)
		in <- pipeline.MakeErrorResult[int](errors.New("error2"))
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 successful results, got %d", len(results))
	}

	if atomic.LoadInt64(&errorCount) != 2 {
		t.Errorf("Expected 2 errors, got %d", atomic.LoadInt64(&errorCount))
	}
}

func TestThroughput(t *testing.T) {
	in := make(chan pipeline.Result[int])

	var infoCount int64
	out := pipeline.Throughput(in, func(info pipeline.ThroughputInfo) {
		atomic.AddInt64(&infoCount, 1)
	})

	go func() {
		defer close(in)
		for i := 0; i < 10; i++ {
			if i%3 == 0 {
				in <- pipeline.MakeErrorResult[int](errors.New("error"))
			} else {
				in <- pipeline.MakeSuccessResult(i)
			}
		}
	}()

	var results []pipeline.Result[int]
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}

	if atomic.LoadInt64(&infoCount) != 10 {
		t.Errorf("Expected 10 throughput info calls, got %d", atomic.LoadInt64(&infoCount))
	}
}

func TestRate(t *testing.T) {
	in := make(chan int)

	var infoCount int64
	out := pipeline.Rate(in, func(info pipeline.RateInfo) {
		atomic.AddInt64(&infoCount, 1)
	})

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	if atomic.LoadInt64(&infoCount) != 5 {
		t.Errorf("Expected 5 rate info calls, got %d", atomic.LoadInt64(&infoCount))
	}
}

func TestFirst(t *testing.T) {
	in := make(chan int)

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	first, ok := pipeline.First(in)
	if !ok {
		t.Error("Expected to get first value")
	}
	if first != 1 {
		t.Errorf("Expected first value 1, got %d", first)
	}
}

func TestLast(t *testing.T) {
	in := make(chan int)

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	last, ok := pipeline.Last(in)
	if !ok {
		t.Error("Expected to get last value")
	}
	if last != 5 {
		t.Errorf("Expected last value 5, got %d", last)
	}
}

func TestToSlice(t *testing.T) {
	in := make(chan int)

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	results := pipeline.ToSlice(in)
	expected := []int{1, 2, 3, 4, 5}

	if len(results) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func TestForEach(t *testing.T) {
	in := make(chan int)

	var sum int64
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		pipeline.ForEach(in, func(x int) {
			atomic.AddInt64(&sum, int64(x))
		})
	}()

	go func() {
		defer close(in)
		for i := 1; i <= 5; i++ {
			in <- i
		}
	}()

	wg.Wait()

	expected := int64(15) // 1+2+3+4+5
	if atomic.LoadInt64(&sum) != expected {
		t.Errorf("Expected sum %d, got %d", expected, atomic.LoadInt64(&sum))
	}
}

func TestPanicRecovery(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic to be recovered, but got: %v", r)
		}
	}()

	// Test invalid buffer size
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid buffer size")
			}
		}()
		in := make(chan int)
		pipeline.Buffer(0, in)
	}()

	// Test invalid worker count
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid worker count")
			}
		}()
		in := make(chan int)
		pipeline.Pool(0, in, func(x int) int { return x })
	}()
}

func BenchmarkTransform(b *testing.B) {
	in := make(chan int, 1000)
	out := pipeline.Transform(in, func(x int) int { return x * 2 })

	b.ResetTimer()

	go func() {
		defer close(in)
		for i := 0; i < b.N; i++ {
			in <- i
		}
	}()

	for range out {
		// Just consume the output
	}
}

func BenchmarkPool(b *testing.B) {
	in := make(chan int, 1000)
	out := pipeline.Pool(4, in, func(x int) int { return x * 2 })

	b.ResetTimer()

	go func() {
		defer close(in)
		for i := 0; i < b.N; i++ {
			in <- i
		}
	}()

	for range out {
		// Just consume the output
	}
}

func BenchmarkFilter(b *testing.B) {
	in := make(chan int, 1000)
	out := pipeline.Filter(in, func(x int) bool { return x%2 == 0 })

	b.ResetTimer()

	go func() {
		defer close(in)
		for i := 0; i < b.N; i++ {
			in <- i
		}
	}()

	for range out {
		// Just consume the output
	}
}

func TestFrom(t *testing.T) {
	ctx := context.Background()

	// Test From
	values := []int{1, 2, 3, 4, 5}
	out := pipeline.From(ctx, values...)

	results := pipeline.ToSlice(out)

	if len(results) != len(values) {
		t.Errorf("Expected %d items, got %d", len(values), len(results))
	}

	for i, result := range results {
		if result != values[i] {
			t.Errorf("Expected %d, got %d", values[i], result)
		}
	}
}

func TestFromSlice(t *testing.T) {
	ctx := context.Background()

	values := []string{"a", "b", "c"}
	out := pipeline.FromSlice(ctx, values)

	results := pipeline.ToSlice(out)

	if len(results) != len(values) {
		t.Errorf("Expected %d items, got %d", len(values), len(results))
	}

	for i, result := range results {
		if result != values[i] {
			t.Errorf("Expected %s, got %s", values[i], result)
		}
	}
}

func TestRange(t *testing.T) {
	ctx := context.Background()

	out := pipeline.Range(ctx, 0, 5)
	results := pipeline.ToSlice(out)

	expected := []int{0, 1, 2, 3, 4}
	if len(results) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(results))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Expected %d, got %d", expected[i], result)
		}
	}
}

func TestRangeStep(t *testing.T) {
	ctx := context.Background()

	out := pipeline.RangeStep(ctx, 0, 10, 2)
	results := pipeline.ToSlice(out)

	expected := []int{0, 2, 4, 6, 8}
	if len(results) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(results))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Expected %d, got %d", expected[i], result)
		}
	}
}
