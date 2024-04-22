package lock_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	// Setup a redis instance in docker.
	// The address can be obtained from redistest.Addr().
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestLock_Success(t *testing.T) {
	client := newClient(t)

	ch := make(chan bool)
	key := t.Name()

	var events []string
	var wg sync.WaitGroup
	wg.Add(2)

	errs := make(chan error, 2)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {
		defer wg.Done()

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 100 ms.
			time.Sleep(100 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		})
		events = append(events, "worker #1: done")
		if err != nil {
			errs <- err
		}
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ch

		locker := lock.New(client)
		// Lock for 1s minimum, wait for 200ms to acquire lock.
		err := locker.Lock(ctx, key, 1*time.Second, 200*time.Millisecond, func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		})
		events = append(events, "worker #2: done")
		if err != nil {
			errs <- err
		}
	}(t)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	expected := []string{
		"worker #1: lock acquired",
		"worker #1: awake",
		"worker #1: done",
		"worker #2: lock acquired",
		"worker #2: done",
	}

	for i := 0; i < len(expected); i++ {
		want := expected[i]
		got := events[i]
		if want != got {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

// TestLock_WaitTimeout is similar to TestLock_Success, except that the second
// goroutine will fail to acquire the lock.
// The first goroutine holds the lock for 200ms.
// The second goroutine fails to acquire the lock within 100ms.
// The second goroutine fails with error.
func TestLock_WaitTimeout(t *testing.T) {
	client := newClient(t)

	ch := make(chan bool)
	key := t.Name()

	var events []string
	var wg sync.WaitGroup
	wg.Add(2)

	errs := make(chan error, 2)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {
		defer wg.Done()

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 200 ms.
			time.Sleep(200 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		})
		events = append(events, "worker #1: done")
		if err != nil {
			errs <- err
		}
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ch

		locker := lock.New(client)
		// Lock for 1s minimum, wait for 100ms to acquire lock.
		err := locker.Lock(ctx, key, 1*time.Second, 100*time.Millisecond, func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		})
		events = append(events, "worker #2: done")
		if err != nil {
			errs <- err
		}
	}(t)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil && !errors.Is(err, lock.ErrLockWaitTimeout) {
			t.Fatalf("want ErrLockWaitTimeout, got %v", err)
		}
	}

	expected := []string{
		"worker #1: lock acquired",
		"worker #2: done",
		"worker #1: awake",
		"worker #1: done",
	}

	for i := 0; i < len(expected); i++ {
		want := expected[i]
		got := events[i]
		if want != got {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

func TestLock_KeyReleased_Timeout(t *testing.T) {
	client := newClient(t)
	errs := make(chan error)
	ch := make(chan bool)
	key := t.Name()

	go func() {
		defer close(errs)

		// Goroutine 1 holds the lock for 200ms.
		locker := lock.New(client)
		errs <- locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
			close(ch)
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}()

	<-ch
	locker := lock.New(client)
	err := locker.Lock(ctx, key, 1*time.Second, 100*time.Millisecond, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, lock.ErrLockWaitTimeout) {
		t.Fatalf("want ErrLockWaitTimeout, got %v", err)
	}

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	testKeyReleased(t, client, key)
}

func TestLock_KeyReleased_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(ctx)

	client := newClient(t)
	locker := lock.New(client)
	key := t.Name()
	err := locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
		cancel()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	testKeyReleased(t, client, key)
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.FlushAll(ctx)
		client.Close()
	})

	return client
}

func testKeyReleased(t *testing.T, client *redis.Client, key string) {
	t.Helper()

	exists, err := client.Exists(context.Background(), key).Result()
	if err != nil {
		t.Fatal(err)
	}
	if exists == 1 {
		t.Fatal("want exists to be false, got true")
	}
}
