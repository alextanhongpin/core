package lock_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestLock_WaitSuccess(t *testing.T) {
	var (
		ch     = make(chan bool)
		client = redistest.Client(t)
		events []string
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	wg.Add(2)
	go func() {
		defer wg.Done()

		var (
			lockTTL = time.Second
			locker  = lock.New(client)
			waitTTL = time.Second
		)

		// Lock 1 will spend 100ms on the work, and release the lock.
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 100 ms.
			time.Sleep(100 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		}, lockTTL, waitTTL)
		is.NoError(err)

		events = append(events, "worker #1: done")
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ch

		locker := lock.New(client)
		lockTTL := 1 * time.Second
		waitTTL := 200 * time.Millisecond

		// Lock 2 will acquire the lock after 100ms.
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		}, lockTTL, waitTTL)
		events = append(events, "worker #2: done")
		is.NoError(err)
	}(t)

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
		client = redistest.Client(t)
		events []string
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)
	wg.Add(2)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {
		defer wg.Done()

		locker := lock.New(client)
		lockTTL := time.Second
		waitTTL := time.Second
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			// Start the second goroutine.
			events = append(events, "worker #1: lock acquired")
			close(ch)

			// Hold for 200 ms.
			time.Sleep(200 * time.Millisecond)

			events = append(events, "worker #1: awake")
			return nil
		}, lockTTL, waitTTL)
		events = append(events, "worker #1: done")
		is.NoError(err)
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ch

		locker := lock.New(client)
		lockTTL := time.Second
		waitTTL := 100 * time.Millisecond
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			events = append(events, "worker #2: lock acquired")
			return nil
		}, lockTTL, waitTTL)
		events = append(events, "worker #2: done")
		is.ErrorIs(err, lock.ErrLockWaitTimeout)
	}(t)

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
		ch     = make(chan bool)
		client = redistest.Client(t)
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Goroutine 1 holds the lock for 100ms.
		locker := lock.New(client)
		lockTTL := time.Second
		waitTTL := time.Second
		err := locker.Do(ctx, key, func(ctx context.Context) error {
			close(ch) // Signal the second goroutine to start.

			time.Sleep(100 * time.Millisecond)
			return nil
		}, lockTTL, waitTTL)
		is.NoError(err)
	}()

	<-ch

	var (
		locker  = lock.New(client)
		lockTTL = time.Second
		waitTTL = time.Duration(0)
	)
	err := locker.Do(ctx, key, func(ctx context.Context) error {
		return nil
	}, lockTTL, waitTTL)
	is.ErrorIs(err, lock.ErrLocked)
}

func TestLock_Unlock_ContextCanceled(t *testing.T) {
	var (
		client  = redistest.Client(t)
		is      = assert.New(t)
		key     = t.Name()
		lockTTL = time.Second
		locker  = lock.New(client)
		waitTTL = time.Second
	)

	ctx, cancel := context.WithCancel(ctx)
	err := locker.Do(ctx, key, func(ctx context.Context) error {
		cancel()
		return nil
	}, lockTTL, waitTTL)
	is.ErrorIs(err, context.Canceled)

	assertNoKey(t, client, key)
}

func TestLock_Unlock_Error(t *testing.T) {
	var (
		client  = redistest.Client(t)
		is      = assert.New(t)
		key     = t.Name()
		lockTTL = time.Second
		locker  = lock.New(client)
		waitTTL = time.Second
	)

	var wantErr = errors.New("want error")
	err := locker.Do(ctx, key, func(ctx context.Context) error {
		return wantErr
	}, lockTTL, waitTTL)
	is.ErrorIs(err, wantErr)

	assertNoKey(t, client, key)
}

func TestLock_Unlock_Deleted(t *testing.T) {
	// Test the scenario where the redis restarts and the key is deleted.
	var (
		ch      = make(chan bool)
		client  = redistest.Client(t)
		is      = assert.New(t)
		key     = t.Name()
		lockTTL = 100 * time.Millisecond
		locker  = lock.New(client)
		waitTTL = time.Second
	)

	go func() {
		<-ch
		status, err := client.Del(ctx, key).Result()
		is.NoError(err)
		is.Equal(int64(1), status)
	}()

	err := locker.Do(ctx, key, func(ctx context.Context) error {
		// Lock acquired. Signal deletion.
		ch <- true
		// Sleep for 2x the lock ttl duration.
		time.Sleep(2 * lockTTL)
		return nil
	}, lockTTL, waitTTL)
	is.ErrorIs(err, lock.ErrConflict)
}

func TestLock_Extend_Success(t *testing.T) {
	var (
		ch     = make(chan bool)
		client = redistest.Client(t)
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()

		locker := lock.New(client)
		lockTTL := 100 * time.Millisecond
		waitTTL := 0 * time.Millisecond

		err := locker.Do(ctx, key, func(ctx context.Context) error {
			// Signal the second goroutine to start.
			close(ch)

			// Holds the lock for 1s. The lock will refresh every 7/10 of 100ms.
			time.Sleep(1 * time.Second)
			return nil
		}, lockTTL, waitTTL)
		is.NoError(err)
	}()

	go func() {
		defer wg.Done()

		// Wait for the signal from the first goroutine.
		<-ch

		locker := lock.New(client)
		lockTTL := 100 * time.Millisecond
		waitTTL := 0 * time.Millisecond

		for i := 1; i < 10; i++ {
			// Try to obtain the lock every 100ms. Because the lock is still held by
			// the first goroutine, it is expected to fail.
			time.Sleep(100 * time.Millisecond)
			err := locker.Do(ctx, key, func(ctx context.Context) error {
				return nil
			}, lockTTL, waitTTL)
			is.ErrorIs(err, lock.ErrLocked)
		}
	}()

	wg.Wait()

	assertNoKey(t, client, key)
}

func TestLock_DoTimeout(t *testing.T) {
	var (
		client  = redistest.Client(t)
		is      = assert.New(t)
		key     = t.Name()
		lockTTL = 50 * time.Millisecond
		locker  = lock.New(client)
		waitTTL = time.Second
	)

	var wantErr = errors.New("want error")
	err := locker.DoTimeout(ctx, key, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)

		return wantErr
	}, lockTTL, waitTTL)
	is.ErrorIs(err, lock.ErrLockTimeout)

	time.Sleep(5 * time.Millisecond) // Ensure the TTL is expired.
	assertNoKey(t, client, key)
}

func assertNoKey(t *testing.T, client *redis.Client, key string) {
	t.Helper()

	_, err := client.Get(context.Background(), key).Result()
	is := assert.New(t)
	is.ErrorIs(err, redis.Nil, "expected key to be deleted")
}
