package promise_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

func TestPromiseWithContext(t *testing.T) {
	ctx := t.Context()

	counter := atomic.Int32{}
	p := promise.New(ctx, func(ctx context.Context) (int, error) {
		counter.Add(1)
		select {
		case <-ctx.Done():
			return 0, context.Cause(ctx)
		case <-time.After(10 * time.Millisecond):
			return 42, nil
		}
	})

	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			res, err := p.Await()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != 42 {
				t.Fatalf("want 42, got %d", res)
			}
			if counter.Load() != 1 {
				t.Fatalf("want function to be called once, was called %d times", counter.Load())
			}
		})
	}
	wg.Wait()
}

func TestPromiseContextCancellation(t *testing.T) {
	ctx := t.Context()
	p := promise.New(ctx, func(ctx context.Context) (int, error) {
		select {
		case <-ctx.Done():
			return 0, context.Cause(ctx)
		case <-time.After(100 * time.Millisecond):
			return 42, nil
		}
	})

	// Cancel context before promise completes
	p.Abort(errors.ErrUnsupported)

	res, err := p.Await()
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatal("want error due to context cancellation")
	}
	if res != 0 {
		t.Fatalf("want 0 res on error, got %d", res)
	}
}

func TestPromiseAbort(t *testing.T) {
	ctx := t.Context()
	p := promise.New(ctx, func(context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})
	p.Abort(errors.ErrUnsupported)

	res, err := p.Await()
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("want unsupported error, got %v", err)
	}
	if res != 0 {
		t.Fatalf("want 0 res on timeout, got %d", res)
	}
}

func TestPromiseStatus(t *testing.T) {
	// Test pending promise
	ctx := t.Context()
	p := promise.New(ctx, func(context.Context) (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	})

	if p.Status() != promise.StatusPending {
		t.Fatal("want promise to be pending")
	}

	// Wait for completion
	res, err := p.Await()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != 42 {
		t.Fatalf("want 42, got %d", res)
	}

	if want, got := promise.StatusFulfilled, p.Status(); want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestPromiseRejectedState(t *testing.T) {
	ctx := t.Context()
	p := promise.New(ctx, func(context.Context) (int, error) {
		return 0, errors.ErrUnsupported
	})

	res, err := p.Await()
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("want test error, got %v", err)
	}
	if res != 0 {
		t.Fatalf("want 0 res on error, got %d", res)
	}

	if p.Status() != promise.StatusRejected {
		t.Fatal("want promise not to be rejected")
	}
}

func TestPromiseAll(t *testing.T) {
	ctx := t.Context()
	res, err := promise.All(
		promise.Resolve(ctx, 1),
		promise.Resolve(ctx, 2),
		promise.Resolve(ctx, 3),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("want 3 res, got %d", len(res))
	}
	for i, want := range []int{1, 2, 3} {
		if res[i] != want {
			t.Fatalf("want res[%d] = %d, got %d", i, want, res[i])
		}
	}
}

func TestPromiseAllWithError(t *testing.T) {
	ctx := t.Context()
	res, err := promise.All(
		promise.Resolve(ctx, 1),
		promise.Reject[int](ctx, errors.ErrUnsupported),
		promise.Resolve(ctx, 3),
	)

	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("want %v, got %v", errors.ErrUnsupported, err)
	}
	if res != nil {
		t.Fatal("want nil res on error")
	}
}

func TestPromisesAllSettled(t *testing.T) {
	ctx := t.Context()
	res := promise.AllSettled(
		promise.Resolve(ctx, 1),
		promise.Reject[int](ctx, errors.ErrUnsupported),
		promise.Resolve(ctx, 3),
	)

	if len(res) != 3 {
		t.Fatalf("want 3 res, got %d", len(res))
	}

	// First promise should be resolved
	if res[0].Status != promise.StatusFulfilled {
		t.Fatal("want first res to be resolved")
	}
	if res[0].Data != 1 {
		t.Fatalf("want first res data to be 1, got %d", res[0].Data)
	}

	// Second promise should be rejected
	if res[1].Status != promise.StatusRejected {
		t.Fatal("want second res to be rejected")
	}
	if !errors.Is(res[1].Error, errors.ErrUnsupported) {
		t.Fatalf("want second res error to be test error, got %v", res[1].Error)
	}

	// Third promise should be resolved
	if res[2].Status != promise.StatusFulfilled {
		t.Fatal("want third res to be resolved")
	}
	if res[2].Data != 3 {
		t.Fatalf("want third res data to be 3, got %d", res[2].Data)
	}
}

func TestPromisesRace(t *testing.T) {
	ctx := t.Context()
	res, err := promise.Race(
		promise.New(ctx, func(context.Context) (int, error) {
			time.Sleep(20 * time.Millisecond)
			return 1, nil
		}),
		promise.New(ctx, func(context.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 2, nil
		}),
		promise.New(ctx, func(context.Context) (int, error) {
			time.Sleep(30 * time.Millisecond)
			return 3, nil
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != 2 {
		t.Fatalf("want fastest promise (2), got %d", res)
	}
}

func TestPromisesAny(t *testing.T) {
	ctx := t.Context()
	res, err := promise.Any(
		promise.Reject[int](ctx, errors.ErrUnsupported),
		promise.New(ctx, func(context.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return 2, nil
		}),
		promise.Reject[int](ctx, errors.ErrUnsupported),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != 2 {
		t.Fatalf("want successful promise (2), got %d", res)
	}
}

func TestPromisesAnyAllRejected(t *testing.T) {
	ctx := t.Context()
	res, err := promise.Any(
		promise.Reject[int](ctx, errors.ErrUnsupported),
		promise.Reject[int](ctx, errors.ErrUnsupported),
		promise.Reject[int](ctx, errors.ErrUnsupported),
	)
	if err == nil {
		t.Fatal("want error when all promises are rejected")
	}
	if res != 0 {
		t.Fatalf("want 0 res when all rejected, got %d", res)
	}

	uw, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatal("want joined errors")
	}
	errSlice := uw.Unwrap()
	for _, err := range errSlice {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("want error, got %v", err)
		}
	}
}

func TestMap(t *testing.T) {
	m := promise.NewMap[string, int]()
	ctx := t.Context()

	var counter atomic.Int32
	var wg sync.WaitGroup
	n := 10
	ch := make(chan struct{})

	for range n {
		wg.Go(func() {
			res, err := m.Do(ctx, t.Name(), func(ctx context.Context) (int, error) {
				<-ch
				counter.Add(1)
				return 42, nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if res != 42 {
				t.Errorf("want 42, got %d", res)
			}
		})
	}

	close(ch)
	wg.Wait()
	if counter.Load() != 1 {
		t.Fatalf("want function to be called once, was called %d times", counter.Load())
	}
}

func TestMapClear(t *testing.T) {
	ctx := t.Context()
	m := promise.NewMap[string, int]()

	// Add some promises
	n := 3
	for i := range n {
		key := fmt.Sprintf("key:%d", i)
		m.Do(ctx, key, func(ctx context.Context) (int, error) {
			return 42, nil
		})
	}
	if m.Size() != 3 {
		t.Fatalf("want 3 promises, got %d", m.Size())
	}

	m.Clear()

	if m.Size() != 0 {
		t.Fatalf("want 0 promises after clear, got %d", m.Size())
	}
}
