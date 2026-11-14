package cache_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/require"
)

// A negative TTL must mean "no expiry" like File and FS, not go-redis KEEPTTL.
func TestRedisNegativeTTLClearsExpiry(t *testing.T) {
	c := cache.MustNewRedis(newClient(t))
	ctx := t.Context()
	require.NoError(t, c.Store(ctx, "k", []byte("a"), time.Hour))
	require.NoError(t, c.Store(ctx, "k", []byte("b"), -1))
	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, time.Duration(-1), ttl)

	require.NoError(t, c.Store(ctx, "k", []byte("b"), time.Hour))
	require.NoError(t, c.CompareAndSwap(ctx, "k", []byte("b"), []byte("c"), -1))
	ttl, err = c.TTL(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, time.Duration(-1), ttl)
}

// Concurrent LoadOrCreate callers must not share one backing slice.
func TestRedisLoadOrCreateReturnsCopies(t *testing.T) {
	c := cache.MustNewRedis(newClient(t))
	create := func(context.Context, string) ([]byte, time.Duration, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("value"), time.Minute, nil
	}
	var wg sync.WaitGroup
	results := make([][]byte, 8)
	for i := range results {
		wg.Go(func() {
			v, _, err := c.LoadOrCreate(t.Context(), t.Name(), create)
			require.NoError(t, err)
			results[i] = v
		})
	}
	wg.Wait()
	for i := range results {
		for j := range i {
			require.NotSame(t, &results[i][0], &results[j][0])
		}
	}
	_, _, err := c.LoadOrCreate(t.Context(), "nil-create", nil)
	require.Error(t, err)
}

// Running out of Wait must be recognisable as ErrLocked.
func TestLockWaitTimeoutIsErrLocked(t *testing.T) {
	l := cache.MustNewLock(newClient(t))
	ctx := t.Context()
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- l.AdvisoryLock(ctx, "k", cache.AdvisoryLockConfig{Do: func(context.Context, string, []byte) error {
			close(held)
			<-release
			return nil
		}})
	}()
	<-held
	err := l.AdvisoryLock(ctx, "k", cache.AdvisoryLockConfig{Wait: 30 * time.Millisecond, Do: func(context.Context, string, []byte) error { return nil }})
	require.ErrorIs(t, err, cache.ErrLocked)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	close(release)
	require.NoError(t, <-done)
}

// Losing a lease (renewal fails) must be recognisable as ErrLocked.
func TestLockLostLeaseIsErrLocked(t *testing.T) {
	client := newClient(t)
	l := cache.MustNewLock(client)
	ctx := t.Context()
	err := l.AdvisoryLock(ctx, "k", cache.AdvisoryLockConfig{Lock: 100 * time.Millisecond, RefreshRatio: .3, Do: func(work context.Context, _ string, _ []byte) error {
		keys, err := client.Keys(ctx, "cache:lease:*").Result()
		require.NoError(t, err)
		require.NoError(t, client.Del(ctx, keys...).Err())
		<-work.Done()
		return nil
	}})
	require.ErrorIs(t, err, cache.ErrLocked)

	// Without renewal, running past the lease also reports ErrLocked.
	_, _, err = l.LoadOrCreate(ctx, "v", cache.LoadOrCreateConfig[[]byte]{Lock: 20 * time.Millisecond, Create: func(context.Context, string) ([]byte, time.Duration, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("late"), 0, nil
	}})
	require.ErrorIs(t, err, cache.ErrLocked)
}

// FuncConfig.Wait lets concurrent callers share one fill instead of failing.
func TestFuncWaitSharesFill(t *testing.T) {
	var calls atomic.Int32
	fn := cache.MustFunc(func(ctx context.Context, id int) (string, time.Duration, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "user", time.Minute, nil
	}, cache.FuncConfig[int]{
		Lock:  cache.MustNewLock(newClient(t)),
		KeyFn: func(_ context.Context, id int) (string, error) { return "user:1", nil },
		Wait:  5 * time.Second,
	})
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			v, _, err := fn(t.Context(), 1)
			require.NoError(t, err)
			require.Equal(t, "user", v)
		})
	}
	wg.Wait()
	require.EqualValues(t, 1, calls.Load())

	_, err := cache.Func(func(context.Context, int) (string, time.Duration, error) { return "", 0, nil }, cache.FuncConfig[int]{
		Lock:  cache.MustNewLock(newClient(t)),
		KeyFn: func(context.Context, int) (string, error) { return "", nil },
		Wait:  -1,
	})
	require.Error(t, err)
}

func TestGoFuncNilCodecDefaultsToJSON(t *testing.T) {
	fs, err := cache.NewFS(t.TempDir())
	require.NoError(t, err)
	defer fs.Close()
	fn := cache.GoFunc(func(_ context.Context, s string) (string, error) { return s + "!", nil }, fs, nil)
	v, err := fn(t.Context(), "hi")
	require.NoError(t, err)
	require.Equal(t, "hi!", v)
}

func TestNewFSCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	fs, err := cache.NewFS(dir)
	require.NoError(t, err)
	defer fs.Close()
	require.NoError(t, fs.Store(t.Context(), "k", []byte("v"), 0))
	_, err = fs.Load(t.Context(), "missing")
	require.True(t, errors.Is(err, cache.ErrNotExist))
}
