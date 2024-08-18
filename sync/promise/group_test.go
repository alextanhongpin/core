package promise_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alextanhongpin/core/sync/promise"
)

// TestGroup tests that the promise is invoked only once
// while other references waits for the promise to be
// resolved/rejected.
func TestGroupResolve(t *testing.T) {
	counter := new(atomic.Int64)
	fn := func() (int, error) {
		counter.Add(1)
		return 42, nil
	}

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	g := promise.NewGroup[int]()

	for range n {
		go func() {
			defer wg.Done()

			n, err := g.Do(t.Name(), fn)
			if err != nil {
				t.Error(err)
			}
			if want, got := 42, n; want != got {
				t.Errorf("want %d, got %d", want, got)
			}
		}()
	}

	wg.Wait()
	if want, got := int64(1), counter.Load(); want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestGroupReject(t *testing.T) {
	wantErr := errors.New("want error")

	counter := new(atomic.Int64)
	fn := func() (int, error) {
		counter.Add(1)
		return 0, wantErr
	}

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	g := promise.NewGroup[int]()

	for range n {
		go func() {
			defer wg.Done()

			_, err := g.Do(t.Name(), fn)
			if !errors.Is(err, wantErr) {
				t.Error(err)
			}
		}()
	}

	wg.Wait()
	if want, got := int64(1), counter.Load(); want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}

// TestGroupForget tests that the existing promise is
// rejected first before deleting to prevent goroutine
// leak.
func TestGroupForget(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	g := promise.NewGroup[int]()
	p, loaded := g.LoadOrStore(t.Name())
	if loaded {
		t.Fatal("want promise to be stored")
	}

	p, loaded = g.LoadOrStore(t.Name())
	if !loaded {
		t.Fatal("want promise to be loaded")
	}

	// Wait in a separate goroutine.
	go func() {
		defer wg.Done()

		_, err := p.Await()
		if !errors.Is(err, promise.ErrAborted) {
			t.Error(err)
		}
	}()

	forgotten := g.Forget(t.Name())
	if !forgotten {
		t.Fatal("want promise key to be forgotten")
	}

	wg.Wait()
}
