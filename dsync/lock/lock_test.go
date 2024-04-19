package lock_test

import (
	"context"
	"errors"
	"fmt"
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

func TestLock(t *testing.T) {
	client := newClient(t)

	ok := make(chan bool)
	key := t.Name()

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {
		defer wg.Done()

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
			close(ok)
			// Hold for 100 ms.
			time.Sleep(100 * time.Millisecond)

			t.Log("1. done")
			return nil
		})
		t.Log("1. err", err)
		if err != nil {
			errs <- err
		}
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ok

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 200*time.Millisecond, func(ctx context.Context) error {
			t.Log("2. done")
			return nil
		})
		t.Log("2. err", err)
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
}

func TestLockTimeout(t *testing.T) {
	client := newClient(t)

	ok := make(chan bool)
	key := t.Name()

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var wantErr = errors.New("want")

	go func() {
		defer wg.Done()

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 1*time.Second, func(ctx context.Context) error {
			close(ok)
			// Hold for 300 ms that will cause timeout.
			time.Sleep(300 * time.Millisecond)

			t.Log("1. done")
			return nil
		})
		t.Log("1. err", err)
		if err != nil {
			errs <- err
		}
	}()

	go func(t *testing.T) {
		defer wg.Done()

		// Wait for the first lock to be acquired.
		<-ok

		locker := lock.New(client)
		err := locker.Lock(ctx, key, 1*time.Second, 200*time.Millisecond, func(ctx context.Context) error {
			t.Log("2. done")
			return nil
		})
		t.Log("2. err", err)
		if err != nil {
			errs <- fmt.Errorf("%w: %w", err, wantErr)
		}
	}(t)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			if errors.Is(err, wantErr) && errors.Is(err, lock.ErrLockWaitTimeout) {
				t.Log("expected error", err)
				continue
			}
			t.Fatal(err)
		}
	}
}

func TestLockContextCancelled(t *testing.T) {
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
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.FlushAll(ctx)
		client.Close()
	})

	return client
}
