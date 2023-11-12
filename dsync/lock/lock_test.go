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

	go func() {
		defer wg.Done()

		// Simulate concurrent operations. This request starts at a later time.
		time.Sleep(100 * time.Millisecond)
		err := locker.Do(ctx, "key", 60*time.Second, func(ctx context.Context) error {
			return nil
		})
		a.ErrorIs(err, lock.ErrLocked)
	}()

	go func() {
		defer wg.Done()

		err := locker.Do(ctx, "key", 60*time.Second, func(ctx context.Context) error {
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
		key := "lock-1"
		locker := lock.New(client, "prefix")
		_, err := locker.Lock(ctx, key, 100*time.Millisecond)
		a := assert.New(t)
		a.Nil(err)

		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.ErrorIs(err, lock.ErrLocked)

		time.Sleep(100 * time.Millisecond)
		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.Nil(err)
	})

	t.Run("when unlock success", func(t *testing.T) {
		key := "lock-2"
		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)
		a := assert.New(t)
		a.Nil(err)
		a.Nil(l.Unlock(ctx))

		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.Nil(err)
	})

	t.Run("when unlock failed", func(t *testing.T) {
		key := "unlock-fail"
		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)
		a := assert.New(t)
		a.Nil(err)
		a.Nil(l.Unlock(ctx))
		a.ErrorIs(l.Unlock(ctx), lock.ErrKeyNotFound)
	})

	t.Run("when extend success", func(t *testing.T) {
		key := "lock-3"
		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 100*time.Millisecond)
		a := assert.New(t)
		a.Nil(err)

		time.Sleep(50 * time.Millisecond)
		a.Nil(l.Extend(ctx, 100*time.Millisecond))
		time.Sleep(50 * time.Millisecond)

		_, err = locker.Lock(ctx, key, 100*time.Millisecond)
		a.ErrorIs(err, lock.ErrLocked)
	})

	t.Run("when extend failed", func(t *testing.T) {
		key := "lock-4"
		locker := lock.New(client, "prefix")
		l, err := locker.Lock(ctx, key, 10*time.Millisecond)
		a := assert.New(t)
		a.Nil(err)

		time.Sleep(50 * time.Millisecond)
		a.ErrorIs(l.Extend(ctx, 100*time.Millisecond), lock.ErrKeyNotFound)
	})
}
