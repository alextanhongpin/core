package lock_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/retry"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestLock_WaitSuccess(t *testing.T) {
	var (
		ch     = make(chan bool)
		events []string
		mu     sync.Mutex
		is     = assert.New(t)
		wg     sync.WaitGroup
	)

	addEvent := func(s string) {
		mu.Lock()
		events = append(events, s)
		mu.Unlock()
	}

	wg.Go(func() {
		// Lock 1 will spend 100ms on the work, and release the lock.
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			// Start the second goroutine.
			addEvent("worker #1: lock acquired")
			close(ch)

			// Hold for 100 ms.
			time.Sleep(100 * time.Millisecond)

			addEvent("worker #1: awake")
			return nil
		}, lock.Config{
			LockTTL:      time.Second,
			Retry:        newRetry(time.Second),
			RefreshRatio: 0.7, // Enable refresh to prevent timeout
		})
		is.NoError(err)

		addEvent("worker #1: done")
	})

	wg.Go(func() {
		// WaitTTL for the first lock to be acquired.
		<-ch

		// Lock 2 will acquire the lock after 100ms.
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			addEvent("worker #2: lock acquired")
			return nil
		}, lock.Config{
			LockTTL:      time.Second,
			Retry:        newRetry(200 * time.Millisecond),
			RefreshRatio: 0.7, // Enable refresh to prevent timeout
		})
		addEvent("worker #2: done")
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
		mu     sync.Mutex
		is     = assert.New(t)
		wg     sync.WaitGroup
	)

	addEvent := func(s string) {
		mu.Lock()
		events = append(events, s)
		mu.Unlock()
	}

	wg.Go(func() {
		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			// Start the second goroutine.
			addEvent("worker #1: lock acquired")
			close(ch)

			// Hold for 200 ms.
			time.Sleep(200 * time.Millisecond)

			addEvent("worker #1: awake")
			return nil
		}, lock.Config{
			LockTTL:      time.Second,
			Retry:        newRetry(time.Second),
			RefreshRatio: 0.7,
		})
		addEvent("worker #1: done")
		is.NoError(err)
	})

	wg.Go(func() {
		// WaitTTL for the first lock to be acquired.
		<-ch

		err := runInLock(t, t.Context(), func(ctx context.Context) error {
			addEvent("worker #2: lock acquired")
			return nil
		}, lock.Config{
			LockTTL:      time.Second,
			Retry:        newRetry(100 * time.Millisecond),
			RefreshRatio: 0.7,
		})
		addEvent("worker #2: done")
		is.ErrorIs(err, retry.ErrLimitExceeded)
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
		}, lock.Config{
			LockTTL:      time.Second,
			Retry:        newRetry(time.Second),
			RefreshRatio: 0.7,
		})
		is.NoError(err)
	})

	<-ch

	err := runInLock(t, t.Context(), func(ctx context.Context) error {
		return nil
	}, lock.Config{
		LockTTL:      time.Second,
		Retry:        newRetry(0), // No wait.
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
	}, lock.Config{
		LockTTL:      time.Second,
		Retry:        newRetry(time.Second),
		RefreshRatio: 0.7,
	})
	is.ErrorIs(err, context.Canceled)
	assertNoKey(t)
}

func TestLock_Unlock_Error(t *testing.T) {
	err := runInLock(t, t.Context(), func(ctx context.Context) error {
		return assert.AnError
	}, lock.Config{
		LockTTL:      time.Second,
		Retry:        newRetry(time.Second),
		RefreshRatio: 0.7,
	})
	assert.ErrorIs(t, err, assert.AnError)
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
	}, lock.Config{
		LockTTL:      lockTTL,
		Retry:        newRetry(waitTTL),
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
		}, lock.Config{
			LockTTL:      100 * time.Millisecond,
			Retry:        newRetry(0),
			RefreshRatio: 0.7,
		})
		is.NoError(err)
	})

	wg.Go(func() {
		// WaitTTL for the signal from the first goroutine.
		<-ch

		locker := lock.MustNew(lock.NewClient(client), lock.Config{
			LockTTL:      100 * time.Millisecond,
			Retry:        newRetry(0),
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
		cfg    = lock.Config{
			LockTTL:      1 * time.Second,
			Retry:        newRetry(1 * time.Second),
			RefreshRatio: 0.7,
		}
		locker = lock.MustNew(client, cfg)
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
		cfg    = lock.Config{
			LockTTL:      50 * time.Millisecond,
			RefreshRatio: 0,
			Retry:        newRetry(time.Second),
		}
		locker = lock.MustNew(client, cfg)
	)

	_ = logger
	err := locker.Do(t.Context(), key, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return assert.AnError
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

func runInLock(t *testing.T, ctx context.Context, fn func(context.Context) error, cfg lock.Config) error {
	var (
		rc  = redistest.Client(t)
		key = t.Name()
	)

	client := lock.NewClient(rc)
	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	locker := lock.MustNew(client, cfg)
	_ = logger
	return locker.Do(ctx, key, fn)
}

func newRetry(duration time.Duration) lock.Retry {
	cfg := retry.DefaultConfig()
	cfg.MaxRetries = 1
	if duration == 0 {
		cfg.MaxRetries = 0
	}
	cfg.Backoff = retry.NewConstantBackoff(duration)
	cfg.Throttler = retry.NewNoopThrottler()
	r, _ := retry.New(cfg)
	return r
}

func TestNew_Defaults(t *testing.T) {
	client := lock.NewClient(redistest.Client(t))
	l, err := lock.New(client, lock.Config{})
	assert.NoError(t, err)
	assert.NotNil(t, l)
	_, err = lock.New(nil, lock.Config{})
	assert.Error(t, err)
	cfg := lock.Config{}.WithDefaults()
	assert.Equal(t, 30*time.Second, cfg.LockTTL)
	assert.Equal(t, 0.0, cfg.RefreshRatio)
	assert.NotNil(t, cfg.Retry)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     lock.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     lock.Config{LockTTL: time.Second, RefreshRatio: 0.5},
			wantErr: false,
		},
		{
			name:    "zero LockTTL",
			cfg:     lock.Config{LockTTL: 0, RefreshRatio: 0.5},
			wantErr: true,
		},
		{
			name:    "negative LockTTL",
			cfg:     lock.Config{LockTTL: -time.Second, RefreshRatio: 0.5},
			wantErr: true,
		},
		{
			name:    "negative RefreshRatio",
			cfg:     lock.Config{LockTTL: time.Second, RefreshRatio: -0.1},
			wantErr: true,
		},
		{
			name:    "RefreshRatio equal to 1",
			cfg:     lock.Config{LockTTL: time.Second, RefreshRatio: 1.0},
			wantErr: true,
		},
		{
			name:    "RefreshRatio greater than 1",
			cfg:     lock.Config{LockTTL: time.Second, RefreshRatio: 1.5},
			wantErr: true,
		},
		{
			name:    "zero RefreshRatio (no refresh) is valid",
			cfg:     lock.Config{LockTTL: time.Second, RefreshRatio: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLock_Panic(t *testing.T) {
	rc := redistest.Client(t)
	client := lock.NewClient(rc)

	t.Run("panic with refresh releases lock", func(t *testing.T) {
		key := t.Name()
		locker := lock.MustNew(client, lock.Config{
			LockTTL:      time.Second,
			RefreshRatio: 0.7,
		})

		assert.PanicsWithValue(t, "something went wrong", func() {
			_ = locker.Do(t.Context(), key, func(ctx context.Context) error {
				panic("something went wrong")
			})
		})

		// Ensure lock was unlocked even after panic.
		_, err := client.Get(context.Background(), key).Result()
		assert.ErrorIs(t, err, redis.Nil)
	})

	t.Run("panic without refresh releases lock", func(t *testing.T) {
		key := t.Name()
		locker := lock.MustNew(client, lock.Config{
			LockTTL:      time.Second,
			RefreshRatio: 0,
		})

		assert.PanicsWithValue(t, "critical error", func() {
			_ = locker.Do(t.Context(), key, func(ctx context.Context) error {
				panic("critical error")
			})
		})

		// Ensure lock was unlocked even after panic.
		_, err := client.Get(context.Background(), key).Result()
		assert.ErrorIs(t, err, redis.Nil)
	})
}

func TestFunc(t *testing.T) {
	rc := redistest.Client(t)
	client := lock.NewClient(rc)
	locker := lock.MustNew(client, lock.Config{})

	t.Run("success", func(t *testing.T) {
		fn := func(ctx context.Context, id int) (string, error) {
			return fmt.Sprintf("val-%d", id), nil
		}
		lockedFn := lock.Func(fn, locker, func(ctx context.Context, id int) (string, error) {
			return fmt.Sprintf("key-%d", id), nil
		})

		res, err := lockedFn(t.Context(), 42)
		assert.NoError(t, err)
		assert.Equal(t, "val-42", res)
	})

	t.Run("keyFn error", func(t *testing.T) {
		wantErr := errors.New("key failed")
		fn := func(ctx context.Context, id int) (string, error) {
			return "should-not-run", nil
		}
		lockedFn := lock.Func(fn, locker, func(ctx context.Context, id int) (string, error) {
			return "", wantErr
		})

		res, err := lockedFn(t.Context(), 1)
		assert.ErrorIs(t, err, wantErr)
		assert.Empty(t, res)
	})

	t.Run("fn error preserves partial result", func(t *testing.T) {
		wantErr := errors.New("work failed")
		fn := func(ctx context.Context, id int) (string, error) {
			return "partial-dirty-data", wantErr
		}
		lockedFn := lock.Func(fn, locker, func(ctx context.Context, id int) (string, error) {
			return fmt.Sprintf("key-%d", id), nil
		})

		res, err := lockedFn(t.Context(), 2)
		assert.ErrorIs(t, err, wantErr)
		assert.Equal(t, "partial-dirty-data", res, "caller owns partial results")
	})
}
