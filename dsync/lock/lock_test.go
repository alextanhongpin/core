package lock_test

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var wantErr = errors.New("want error")

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestLock_WaitSuccess(t *testing.T) {
	var (
		ch     = make(chan bool)
		events []string
		is     = assert.New(t)
		wg     sync.WaitGroup
	)

	wg.Go(func() {
		// Lock 1 will spend 100ms on the work, and release the lock.
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 100 ms.
			time.Sleep(100 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		}, &lock.Config{
			LockTTL:      time.Second,
			WaitTTL:      time.Second,
			RefreshRatio: 0.7, // Enable refresh to prevent timeout
		})
		is.NoError(err)

		events = append(events, "worker #1: done")
	})

	wg.Go(func() {
		// WaitTTL for the first lock to be acquired.
		<-ch

		// Lock 2 will acquire the lock after 100ms.
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		}, &lock.Config{
			LockTTL:      time.Second,
			WaitTTL:      200 * time.Millisecond,
			RefreshRatio: 0.7, // Enable refresh to prevent timeout
		})
		events = append(events, "worker #2: done")
		is.NoError(err)
	})

	wg.Wait()
	is.Equal([]string{
		"worker #1: lock acquired",
		"worker #1: awake",
		"worker #1: done",
		"worker #2: lock acquired",
		"worker #2: done",
	}, events)
}

// TestLock_WaitTimeout is similar to TestLock_WaitSuccess, except that the second
// goroutine will fail to acquire the lock.
// The first goroutine holds the lock for 200ms.
// The second goroutine fails to acquire the lock within 100ms.
// The second goroutine fails with error.
func TestLock_WaitTimeout(t *testing.T) {
	var (
		ch     = make(chan bool)
		events []string
		is     = assert.New(t)
		wg     sync.WaitGroup
	)

	wg.Go(func() {
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 200 ms.
			time.Sleep(200 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		}, &lock.Config{
			LockTTL:      time.Second,
			WaitTTL:      time.Second,
			RefreshRatio: 0.7,
		})
		events = append(events, "worker #1: done")
		is.NoError(err)
	})

	wg.Go(func() {
		// WaitTTL for the first lock to be acquired.
		<-ch

		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		}, &lock.Config{
			LockTTL:      time.Second,
			WaitTTL:      100 * time.Millisecond,
			RefreshRatio: 0.7,
		})
		events = append(events, "worker #2: done")
		is.ErrorIs(err, lock.ErrLockWaitTimeout)
	})

	wg.Wait()
	is.Equal([]string{
		"worker #1: lock acquired",
		"worker #2: done",
		"worker #1: awake",
		"worker #1: done",
	}, events)
}

// TestLock_NoWait is similar to TestLock_WaitTimeout, except that the second
// goroutine will fail to acquire the lock.
// The first goroutine holds the lock for 200ms.
// The second goroutine will not wait for the lock.
// The second goroutine fails with error.
func TestLock_NoWait(t *testing.T) {
	var (
		ch = make(chan bool)
		is = assert.New(t)
		wg sync.WaitGroup
	)

	wg.Go(func() {
		// Goroutine 1 holds the lock for 100ms.
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			close(ch) // Signal the second goroutine to start.

			time.Sleep(100 * time.Millisecond)
			return nil
		}, &lock.Config{
			LockTTL:      time.Second,
			WaitTTL:      time.Second,
			RefreshRatio: 0.7,
		})
		is.NoError(err)
	})

	<-ch

	err := runInLock(t, t.Context(), func(ctx context.Context) error {
		return nil
	}, &lock.Config{
		LockTTL:      time.Second,
		WaitTTL:      0, // No wait.
		RefreshRatio: 0.7,
	})
	is.ErrorIs(err, lock.ErrLocked)

	wg.Wait()
}

func TestLock_Unlock_ContextCanceled(t *testing.T) {
	var (
		is  = assert.New(t)
		ctx = t.Context()
	)

	ctx, cancel := context.WithCancel(ctx)
	err := runInLock(t, ctx, func(ctx context.Context) error {
		cancel()
		return nil
	}, &lock.Config{
		LockTTL:      time.Second,
		WaitTTL:      time.Second,
		RefreshRatio: 0.7,
	})
	is.ErrorIs(err, context.Canceled)
	assertNoKey(t)
}

func TestLock_Unlock_Error(t *testing.T) {
	err := runInLock(t, t.Context(), func(ctx context.Context) error {
		return wantErr
	}, &lock.Config{
		LockTTL:      time.Second,
		WaitTTL:      time.Second,
		RefreshRatio: 0.7,
	})
	assert.ErrorIs(t, err, wantErr)
	assertNoKey(t)
}

func TestLock_Unlock_Deleted(t *testing.T) {
	// Test the scenario where the redis restarts and the key is deleted.
	var (
		ch      = make(chan bool)
		client  = redistest.Client(t)
		is      = assert.New(t)
		key     = t.Name()
		lockTTL = 100 * time.Millisecond
		waitTTL = time.Second
	)

	go func() {
		<-ch
		status, err := client.Del(t.Context(), key).Result()
		is.NoError(err)
		is.Equal(int64(1), status)
	}()

	err := runInLock(t, t.Context(), func(ctx context.Context) error {
		// Lock acquired. Signal deletion.
		ch <- true
		// Sleep for 2x the lock ttl duration.
		time.Sleep(2 * lockTTL)
		return nil
	}, &lock.Config{
		LockTTL:      lockTTL,
		WaitTTL:      waitTTL,
		RefreshRatio: 0.5, // Enable extension so it can detect key deletion
	})
	is.ErrorIs(err, lock.ErrLocked)
}

func TestLock_Extend_Success(t *testing.T) {
	var (
		ch     = make(chan bool)
		client = redistest.Client(t)
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)

	wg.Go(func() {
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			// Signal the second goroutine to start.
			close(ch)

			// Holds the lock for 1s. The lock will refresh every 7/10 of 100ms.
			time.Sleep(1 * time.Second)
			return nil
		}, &lock.Config{
			LockTTL:      100 * time.Millisecond,
			WaitTTL:      0,
			RefreshRatio: 0.7,
		})
		is.NoError(err)
	})

	wg.Go(func() {
		// WaitTTL for the signal from the first goroutine.
		<-ch

		locker := lock.New(lock.NewClient(client), &lock.Config{
			LockTTL:      100 * time.Millisecond,
			WaitTTL:      0,
			RefreshRatio: 0.7,
		})

		for i := 1; i < 10; i++ {
			// Try to obtain the lock every 100ms. Because the lock is still held by
			// the first goroutine, it is expected to fail.
			time.Sleep(100 * time.Millisecond)
			err := locker.Do(t.Context(), key, func(ctx context.Context) error {
				return nil
			})
			is.ErrorIs(err, lock.ErrLocked)
		}
	})

	wg.Wait()

	assertNoKey(t)
}

func TestLock_Concurrent(t *testing.T) {
	var (
		client = lock.NewClient(redistest.Client(t))
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
		cfg    = &lock.Config{
			LockTTL:      1 * time.Second,
			WaitTTL:      1 * time.Second,
			RefreshRatio: 0.7,
		}
		locker = lock.New(client, cfg)
		fn     = func(ctx context.Context, a any) (any, error) {
			time.Sleep(rand.N(100 * time.Millisecond))
			return nil, nil
		}
	)

	fn = lock.Func(fn, locker, func(context.Context, any) (string, error) {
		return key, nil
	})

	for range 10 {
		wg.Go(func() {
			_, err := fn(t.Context(), nil)
			is.NoError(err)
		})
	}
	wg.Wait()
}

func TestLock_DoTimeout(t *testing.T) {
	var (
		client = lock.NewClient(redistest.Client(t))
		is     = assert.New(t)
		key    = t.Name()
		logger = slog.New(slog.NewTextHandler(t.Output(), nil))
		cfg    = &lock.Config{
			RefreshRatio: 0,
			LockTTL:      50 * time.Millisecond,
			WaitTTL:      time.Second,
		}
		locker = lock.New(client, cfg)
	)

	locker.Logger = logger
	err := locker.Do(t.Context(), key, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return wantErr
	})
	is.ErrorIs(err, lock.ErrLockTimeout)

	time.Sleep(5 * time.Millisecond) // Ensure the TTL is expired.
	assertNoKey(t)
}

func assertNoKey(t *testing.T) {
	t.Helper()
	var (
		is     = assert.New(t)
		client = redistest.Client(t)
		key    = t.Name()
		_, err = client.Get(context.Background(), key).Result()
	)
	is.ErrorIs(err, redis.Nil, "expected key to be deleted")
}

func runInLock(t *testing.T, ctx context.Context, fn func(context.Context) error, cfg *lock.Config) error {
	var (
		rc  = redistest.Client(t)
		key = t.Name()
	)

	client := lock.NewClient(rc)
	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	locker := lock.New(client, cfg)
	locker.Logger = logger
	return locker.Do(ctx, key, fn)
}
