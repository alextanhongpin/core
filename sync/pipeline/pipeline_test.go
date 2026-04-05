package pipeline_test

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func TestBatch(t *testing.T) {
	in := make(chan int)
	out := pipeline.Batch(in, 3, 100*time.Millisecond)

	go func() {
		defer close(in)
		for i := 1; i <= 7; i++ {
			in <- i
			time.Sleep(10 * time.Millisecond)
		}
	}()

	batches := pipeline.Collect(out)

	// Should get multiple batches
	if len(batches) < 2 {
		t.Errorf("Expected at least 2 batches, got %d", len(batches))
	}

	// First batch should be size 3
	if len(batches[0]) != 3 {
		t.Errorf("Expected first batch size 3, got %d", len(batches[0]))
	}
}

func TestDebounce(t *testing.T) {
	in := make(chan int)
	out := pipeline.Debounce(in, 10*time.Millisecond)

	go func() {
		defer close(in)

		for i := range 5 {
			in <- i
			time.Sleep(3 * time.Millisecond)
		}
	}()
	res := pipeline.Collect(out)
	want := []int{0, 3}
	got := res
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDedup(t *testing.T) {
	in := make(chan int)
	out := pipeline.Dedup(in)

	go func() {
		defer close(in)

		for i := range 3 {
			for _, n := range slices.Repeat([]int{i}, 3) {
				in <- n
			}
		}
	}()
	res := pipeline.Collect(out)
	want := []int{0, 1, 2}
	got := res
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %v, got %v", want, got)
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

func TestFanOut(t *testing.T) {
	in := make(chan int)
	outputs := pipeline.FanOut(in, 3)

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

func TestRateLimit(t *testing.T) {
	in := make(chan int)
	out := pipeline.RateLimit(in, 2, 50*time.Millisecond)

	start := time.Now()
	go func() {
		defer close(in)
		for i := range 4 {
			in <- i
		}
	}()

	res := pipeline.Collect(out)
	elapsed := time.Since(start)
	t.Log(elapsed)

	// Should take at least 150ms for 3 items with 50ms throttle
	if elapsed > 110*time.Millisecond {
		t.Error("not rate limited")
	}

	want := 4
	got := len(res)
	if want != got {
		t.Errorf("want %d, got %d", want, got)
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

func TestFromSlice(t *testing.T) {
	ctx := context.Background()

	values := []string{"a", "b", "c"}
	out := pipeline.SourceSlice(ctx, values)

	results := pipeline.Collect(out)

	if len(results) != len(values) {
		t.Errorf("Expected %d items, got %d", len(values), len(results))
	}

	for i, result := range results {
		if result != values[i] {
			t.Errorf("Expected %s, got %s", values[i], result)
		}
	}
}
