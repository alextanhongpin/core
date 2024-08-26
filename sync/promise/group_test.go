package promise_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
)

// TestGroupResolve tests that the promise is invoked only once
// while other references waits for the promise to be
// resolved.
func TestGroupResolve(t *testing.T) {
	counter := new(atomic.Int64)

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	g := promise.NewGroup[int]()

	for range n {
		go func() {
			defer wg.Done()

			n, err := g.Do(t.Name(), func() (int, error) {
				counter.Add(1)
				return 42, nil
			})
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
	if want, got := 1, g.Len(); want != got {
		t.Fatalf("Len: want %d, got %d", want, got)
	}
}

// TestGroupReject tests that the promise is invoked only once
// while other references waits for the promise to be
// rejected.
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

	if want, got := 1, g.Len(); want != got {
		t.Fatalf("Len: want %d, got %d", want, got)
	}
}

// TestGroupDoForget tests that the cache is cleared on completion.
func TestGroupDoForget(t *testing.T) {
	t.Run("singleflight", func(t *testing.T) {
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		g := promise.NewGroup[int]()

		for range n {
			go func() {
				defer wg.Done()

				n, err := g.DoAndForget(t.Name(), func() (int, error) {
					time.Sleep(50 * time.Millisecond)
					return 42, nil
				})
				if err != nil {
					t.Error(err)
				}
				if want, got := 42, n; want != got {
					t.Errorf("want %d, got %d", want, got)
				}
			}()
		}

		wg.Wait()

		if want, got := 0, g.Len(); want != got {
			t.Fatalf("Len: want %d, got %d", want, got)
		}
	})

	t.Run("has idle promise", func(t *testing.T) {
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		g := promise.NewGroup[int]()
		_, _ = g.LoadOrStore(t.Name())

		for range n {
			go func() {
				defer wg.Done()

				n, err := g.DoAndForget(t.Name(), func() (int, error) {
					time.Sleep(50 * time.Millisecond)
					return 42, nil
				})
				if err != nil {
					t.Error(err)
				}
				if want, got := 42, n; want != got {
					t.Errorf("want %d, got %d", want, got)
				}
			}()
		}

		wg.Wait()

		if want, got := 0, g.Len(); want != got {
			t.Fatalf("Len: want %d, got %d", want, got)
		}
	})

	t.Run("has fulfilled promise", func(t *testing.T) {
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		g := promise.NewGroup[int]()
		p, _ := g.LoadOrStore(t.Name())
		p.Resolve(42)

		for range n {
			go func() {
				defer wg.Done()

				n, err := g.DoAndForget(t.Name(), func() (int, error) {
					time.Sleep(50 * time.Millisecond)
					return 42, nil
				})
				if err != nil {
					t.Error(err)
				}
				if want, got := 42, n; want != got {
					t.Errorf("want %d, got %d", want, got)
				}
			}()
		}

		wg.Wait()

		// NOTE: The key is not deleted if promise is not created by DoAndForget.
		if want, got := 1, g.Len(); want != got {
			t.Fatalf("Len: want %d, got %d", want, got)
		}
	})

	t.Run("has rejected promise", func(t *testing.T) {
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		g := promise.NewGroup[int]()
		p, _ := g.LoadOrStore(t.Name())
		p.Reject(wantErr)

		for range n {
			go func() {
				defer wg.Done()

				_, err := g.DoAndForget(t.Name(), func() (int, error) {
					time.Sleep(50 * time.Millisecond)
					return 42, nil
				})
				if !errors.Is(err, wantErr) {
					t.Fatalf("want %v, got %v", wantErr, err)
				}
			}()
		}

		wg.Wait()

		// NOTE: The key is not deleted if promise is not created by DoAndForget.
		if want, got := 1, g.Len(); want != got {
			t.Fatalf("Len: want %d, got %d", want, got)
		}
	})
}
