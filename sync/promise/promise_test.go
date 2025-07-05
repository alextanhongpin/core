package promise_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func TestPromiseWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := atomic.Int32{}
	p := promise.NewWithContext(ctx, func(ctx context.Context) (int, error) {
		counter.Add(1)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return 42, nil
		}
	})

	result, err := p.Await()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
	if counter.Load() != 1 {
		t.Fatalf("expected function to be called once, was called %d times", counter.Load())
	}
}

func TestPromiseContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	p := promise.NewWithContext(ctx, func(ctx context.Context) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return 42, nil
		}
	})

	// Cancel context before promise completes
	cancel()

	result, err := p.Await()
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
	if result != 0 {
		t.Fatalf("expected 0 result on error, got %d", result)
	}
}

func TestPromiseTimeout(t *testing.T) {
	p := promise.New(func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})

	result, err := p.AwaitWithTimeout(10 * time.Millisecond)
	if err != promise.ErrTimeout {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0 result on timeout, got %d", result)
	}
}

func TestPromiseCancel(t *testing.T) {
	p := promise.New(func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})

	p.Cancel()

	result, err := p.Await()
	if err != promise.ErrCanceled {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0 result on cancellation, got %d", result)
	}
}

func TestPromiseState(t *testing.T) {
	// Test pending promise
	p := promise.New(func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	})

	if !p.IsPending() {
		t.Fatal("expected promise to be pending")
	}
	if p.IsResolved() {
		t.Fatal("expected promise not to be resolved")
	}
	if p.IsRejected() {
		t.Fatal("expected promise not to be rejected")
	}

	// Wait for completion
	result, err := p.Await()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}

	if p.IsPending() {
		t.Fatal("expected promise not to be pending")
	}
	if !p.IsResolved() {
		t.Fatal("expected promise to be resolved")
	}
	if p.IsRejected() {
		t.Fatal("expected promise not to be rejected")
	}
}

func TestPromiseRejectedState(t *testing.T) {
	expectedErr := errors.New("test error")
	p := promise.New(func() (int, error) {
		return 0, expectedErr
	})

	result, err := p.Await()
	if err != expectedErr {
		t.Fatalf("expected test error, got %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0 result on error, got %d", result)
	}

	if p.IsPending() {
		t.Fatal("expected promise not to be pending")
	}
	if p.IsResolved() {
		t.Fatal("expected promise not to be resolved")
	}
	if !p.IsRejected() {
		t.Fatal("expected promise to be rejected")
	}
}

func TestPromisesAll(t *testing.T) {
	promises := promise.Promises[int]{
		promise.Resolve(1),
		promise.Resolve(2),
		promise.Resolve(3),
	}

	results, err := promises.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, expected := range []int{1, 2, 3} {
		if results[i] != expected {
			t.Fatalf("expected results[%d] = %d, got %d", i, expected, results[i])
		}
	}
}

func TestPromisesAllWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	promises := promise.Promises[int]{
		promise.Resolve(1),
		promise.Reject[int](expectedErr),
		promise.Resolve(3),
	}

	results, err := promises.All()
	if err != expectedErr {
		t.Fatalf("expected test error, got %v", err)
	}
	if results != nil {
		t.Fatal("expected nil results on error")
	}
}

func TestPromisesAllSettled(t *testing.T) {
	expectedErr := errors.New("test error")
	promises := promise.Promises[int]{
		promise.Resolve(1),
		promise.Reject[int](expectedErr),
		promise.Resolve(3),
	}

	results := promises.AllSettled()
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// First promise should be resolved
	if !results[0].IsResolved() {
		t.Fatal("expected first result to be resolved")
	}
	if results[0].Data != 1 {
		t.Fatalf("expected first result data to be 1, got %d", results[0].Data)
	}

	// Second promise should be rejected
	if !results[1].IsRejected() {
		t.Fatal("expected second result to be rejected")
	}
	if results[1].Err != expectedErr {
		t.Fatalf("expected second result error to be test error, got %v", results[1].Err)
	}

	// Third promise should be resolved
	if !results[2].IsResolved() {
		t.Fatal("expected third result to be resolved")
	}
	if results[2].Data != 3 {
		t.Fatalf("expected third result data to be 3, got %d", results[2].Data)
	}
}

func TestPromisesRace(t *testing.T) {
	promises := promise.Promises[int]{
		promise.New(func() (int, error) {
			time.Sleep(20 * time.Millisecond)
			return 1, nil
		}),
		promise.New(func() (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 2, nil
		}),
		promise.New(func() (int, error) {
			time.Sleep(30 * time.Millisecond)
			return 3, nil
		}),
	}

	result, err := promises.Race()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 2 {
		t.Fatalf("expected fastest promise (2), got %d", result)
	}
}

func TestPromisesAny(t *testing.T) {
	testErr := errors.New("test error")
	promises := promise.Promises[int]{
		promise.Reject[int](testErr),
		promise.New(func() (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 2, nil
		}),
		promise.Reject[int](testErr),
	}

	result, err := promises.Any()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 2 {
		t.Fatalf("expected successful promise (2), got %d", result)
	}
}

func TestPromisesAnyAllRejected(t *testing.T) {
	testErr := errors.New("test error")
	promises := promise.Promises[int]{
		promise.Reject[int](testErr),
		promise.Reject[int](testErr),
		promise.Reject[int](testErr),
	}

	result, err := promises.Any()
	if err == nil {
		t.Fatal("expected error when all promises are rejected")
	}
	if result != 0 {
		t.Fatalf("expected 0 result when all rejected, got %d", result)
	}
}

func TestMapDoWithContext(t *testing.T) {
	m := promise.NewMap[int]()
	counter := atomic.Int32{}

	var wg sync.WaitGroup
	n := 10
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result, err := m.DoWithContext("test", ctx, func(ctx context.Context) (int, error) {
				counter.Add(1)
				return 42, nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != 42 {
				t.Errorf("expected 42, got %d", result)
			}
		}()
	}

	wg.Wait()

	if counter.Load() != 1 {
		t.Fatalf("expected function to be called once, was called %d times", counter.Load())
	}
}

func TestMapLockWithContext(t *testing.T) {
	m := promise.NewMap[int]()
	counter := atomic.Int32{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := m.LockWithContext("test", ctx, func(ctx context.Context) (int, error) {
		counter.Add(1)
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}

	// Promise should be removed after Lock
	if m.Len() != 0 {
		t.Fatalf("expected map to be empty after Lock, but length is %d", m.Len())
	}
}

func TestMapClear(t *testing.T) {
	m := promise.NewMap[int]()

	// Add some promises
	_, _ = m.LoadOrStore("key1")
	_, _ = m.LoadOrStore("key2")
	_, _ = m.LoadOrStore("key3")

	if m.Len() != 3 {
		t.Fatalf("expected 3 promises, got %d", m.Len())
	}

	m.Clear()

	if m.Len() != 0 {
		t.Fatalf("expected 0 promises after clear, got %d", m.Len())
	}
}

func TestPoolWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := promise.NewPoolWithContext[int](ctx, 2)
	counter := atomic.Int32{}

	// Add multiple tasks
	for i := 0; i < 5; i++ {
		err := pool.DoWithContext(ctx, func(ctx context.Context) (int, error) {
			counter.Add(1)
			time.Sleep(10 * time.Millisecond)
			return int(counter.Load()), nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	results, err := pool.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	if counter.Load() != 5 {
		t.Fatalf("expected 5 function calls, got %d", counter.Load())
	}
}

func TestPoolCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := promise.NewPoolWithContext[int](ctx, 1) // Use limit of 1 to ensure blocking

	// Start a long-running task that will block the pool
	err := pool.DoWithContext(context.Background(), func(ctx context.Context) (int, error) {
		time.Sleep(200 * time.Millisecond) // Long enough to block
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel the pool's context
	cancel()

	// Wait for cancellation to propagate
	time.Sleep(100 * time.Millisecond)

	// Try to add another task - should fail because pool context is canceled
	// This should return immediately with ErrCanceled
	err = pool.DoWithContext(context.Background(), func(ctx context.Context) (int, error) {
		return 1, nil
	})
	if err != promise.ErrCanceled {
		t.Fatalf("expected canceled error, got %v", err)
	}
}

func TestNilFunctionHandling(t *testing.T) {
	// Test New with nil function
	p := promise.New[int](nil)
	result, err := p.Await()
	if err != promise.ErrNilFunction {
		t.Fatalf("expected nil function error, got %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0 result, got %d", result)
	}

	// Test NewWithContext with nil function
	p2 := promise.NewWithContext[int](context.Background(), nil)
	result2, err2 := p2.Await()
	if err2 != promise.ErrNilFunction {
		t.Fatalf("expected nil function error, got %v", err2)
	}
	if result2 != 0 {
		t.Fatalf("expected 0 result, got %d", result2)
	}
}

func TestPromisePanicRecovery(t *testing.T) {
	p := promise.New(func() (int, error) {
		panic("test panic")
	})

	result, err := p.Await()
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if result != 0 {
		t.Fatalf("expected 0 result on panic, got %d", result)
	}
}

func TestEmptyPromisesEdgeCases(t *testing.T) {
	var promises promise.Promises[int]

	// Test All with empty slice
	results, err := promises.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}

	// Test AllSettled with empty slice
	settled := promises.AllSettled()
	if len(settled) != 0 {
		t.Fatalf("expected empty settled results, got %d", len(settled))
	}

	// Test Race with empty slice
	_, err = promises.Race()
	if err != promise.ErrEmptyPromises {
		t.Fatalf("expected empty promises error, got %v", err)
	}

	// Test Any with empty slice
	_, err = promises.Any()
	if err != promise.ErrEmptyPromises {
		t.Fatalf("expected empty promises error, got %v", err)
	}
}
