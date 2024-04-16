package lock_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
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

func TestDo(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})

	locker := lock.New(client, "prefix")

	a := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(2)

	// Simulate concurrent worker 1.
	go func() {
		defer wg.Done()

		// Delay the first worker to get it locked.
		time.Sleep(100 * time.Millisecond)
		err := locker.Do(ctx, "key", 60*time.Second, func(ctx context.Context) error {
			return nil
		})
		a.ErrorIs(err, lock.ErrLocked)
	}()

	// Similate concurrent worker 2.
	go func() {
		defer wg.Done()

		err := locker.Do(ctx, "key", 60*time.Second, func(ctx context.Context) error {
			// Simulate work that holds the lock.
			time.Sleep(200 * time.Millisecond)
			return nil
		})
		a.Nil(err)
	}()

	wg.Wait()
}

func TestLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})

	t.Run("when locked", func(t *testing.T) {
		key := t.Name()
		locker := lock.New(client, "prefix")
		_, err := locker.Lock(ctx, key, 100*time.Millisecond)

		a := assert.New(t)
		a.Nil(err, "expected error to be nil indicating lock is acquired")

		// When a separate process attempts to lock the same key.
		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.ErrorIs(err, lock.ErrLocked, "expected error to be ErrLocked")

		// When the lock expires.
		time.Sleep(100 * time.Millisecond)
		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.Nil(err, "expected error to be nil indicating lock is acquired")
	})

	t.Run("when unlock success", func(t *testing.T) {
		key := t.Name()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)

		a := assert.New(t)
		a.Nil(err, "expected error to be nil indicating lock is acquired")

		err = l.Unlock(ctx)
		a.Nil(err, "expected error to be nil indicating lock is released")

		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.Nil(err, "expected error to be nil indicating lock is acquired")
	})

	t.Run("when unlock failed", func(t *testing.T) {
		key := t.Name()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)

		a := assert.New(t)
		a.Nil(err, "expected error to be nil indicating lock is acquired")

		err = l.Unlock(ctx)
		a.Nil(err, "expected error to be nil indicating lock is released")

		err = l.Unlock(ctx)
		a.ErrorIs(err, lock.ErrKeyNotFound, "expected error to be ErrKeyNotFound indicating lock has been released")
	})

	t.Run("when extend success", func(t *testing.T) {
		key := t.Name()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)

		a := assert.New(t)
		a.Nil(err, "expected error to be nil indicating lock has been acquired")

		// Sleep for 50ms so that only 50ms remains.
		time.Sleep(50 * time.Millisecond)

		err = l.Extend(ctx, 100*time.Millisecond)
		a.Nil(err, "expected error to be nil indicating lock has been extended for 100ms")

		// Sleep for another 50ms to proof the lock has been extended.
		time.Sleep(50 * time.Millisecond)

		// Simulate another process trying to acquire lock.
		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.ErrorIs(err, lock.ErrLocked, "expected error to be ErrLocked indicating the key is still locked")
	})

	t.Run("when extend failed", func(t *testing.T) {
		key := t.Name()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 10*time.Millisecond)

		a := assert.New(t)
		a.Nil(err, "expected error to be nil indicating lock is acquired")

		// Sleep for 50ms to expire the lock.
		time.Sleep(50 * time.Millisecond)
		err = l.Extend(ctx, 100*time.Millisecond)
		a.ErrorIs(err, lock.ErrKeyNotFound, "expected error to be ErrKeyNotFound indicating that the process no longer holds exclusive rights to the lock")
	})
}

func TestLockWait(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})

	ok := make(chan bool)
	key := "lock_wait"

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 1*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		// Allow the second goroutine to start.
		close(ok)

		// Hold for 100 ms.
		time.Sleep(100 * time.Millisecond)
		l.Unlock(ctx)
	}()

	go func() {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ok

		locker := lock.New(client, "prefix")
		l, err := locker.LockWait(ctx, key, 100*time.Millisecond, 200*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Unlock(ctx)
	}()
	wg.Wait()
}

func TestDoWait(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})

	ok := make(chan bool)
	key := "lock_wait"

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 1*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		// Allow the second goroutine to start.
		close(ok)

		// Hold for 100 ms.
		time.Sleep(100 * time.Millisecond)
		l.Unlock(ctx)
	}()

	go func() {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ok

		locker := lock.New(client, "prefix")
		err := locker.DoWait(ctx, key, 100*time.Millisecond, 200*time.Millisecond, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}()
	wg.Wait()
}
