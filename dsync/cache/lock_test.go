package cache_test

import (
	"context"
	"fmt"
	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLockPreservesArbitraryValues(t *testing.T) {
	client := newClient(t)
	ctx := t.Context()
	key := t.Name()
	require.NoError(t, client.Set(ctx, key, "lock:valid-data", 0).Err())
	l := cache.NewLock(client)
	v, loaded, err := l.LoadOrCreate(ctx, key, &cache.LoadOrCreateConfig[[]byte]{Create: func(context.Context, string) ([]byte, time.Duration, error) {
		t.Fatal("cached value should bypass factory")
		return nil, 0, nil
	}})
	require.NoError(t, err)
	require.True(t, loaded)
	require.Equal(t, []byte("lock:valid-data"), v)
}
func TestLockRenewalAndAdmission(t *testing.T) {
	l := cache.NewLock(newClient(t))
	ctx := t.Context()
	key := t.Name()
	started := make(chan struct{})
	finish := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- l.AdvisoryLock(ctx, key, &cache.AdvisoryLockConfig{Lock: 100 * time.Millisecond, RefreshRatio: .5, Do: func(work context.Context, _ string, _ []byte) error {
			close(started)
			select {
			case <-finish:
				return nil
			case <-work.Done():
				return context.Cause(work)
			}
		}})
	}()
	<-started
	time.Sleep(250 * time.Millisecond)
	err := l.AdvisoryLock(ctx, key, &cache.AdvisoryLockConfig{Wait: 30 * time.Millisecond, Do: func(context.Context, string, []byte) error { t.Error("admitted while leader active"); return nil }})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	close(finish)
	require.NoError(t, <-result)
	require.NoError(t, l.AdvisoryLock(ctx, key, &cache.AdvisoryLockConfig{Do: func(context.Context, string, []byte) error { return nil }}))
}
func TestLostLeaseCannotPublish(t *testing.T) {
	client := newClient(t)
	l := cache.NewLock(client)
	ctx := t.Context()
	key := t.Name()
	_, _, err := l.LoadOrCreate(ctx, key, &cache.LoadOrCreateConfig[[]byte]{Lock: 20 * time.Millisecond, Create: func(context.Context, string) ([]byte, time.Duration, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("stale"), time.Minute, nil
	}})
	require.Error(t, err)
	require.EqualValues(t, 0, client.Exists(ctx, key).Val())
}
func TestLockPanicReleasesLease(t *testing.T) {
	l := cache.NewLock(newClient(t))
	key := t.Name()
	require.Panics(t, func() {
		_ = l.AdvisoryLock(t.Context(), key, &cache.AdvisoryLockConfig{Do: func(context.Context, string, []byte) error { panic("boom") }})
	})
	require.NoError(t, l.AdvisoryLock(t.Context(), key, &cache.AdvisoryLockConfig{Do: func(context.Context, string, []byte) error { return nil }}))
}
func TestStorageFactoryReentrancy(t *testing.T) {
	file, err := cache.NewFile(t.TempDir() + "/cache.json")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	fs, err := cache.NewFS(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = fs.Close() })
	for i, store := range []interface {
		LoadOrCreate(context.Context, string, func(context.Context, string) ([]byte, time.Duration, error)) ([]byte, bool, error)
		Store(context.Context, string, []byte, time.Duration) error
	}{file, fs} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, _, err := store.LoadOrCreate(t.Context(), "outer", func(ctx context.Context, _ string) ([]byte, time.Duration, error) {
				return []byte("outer"), 0, store.Store(ctx, "inner", []byte("inner"), 0)
			})
			require.NoError(t, err)
		})
	}
}
func TestRedisExpiryPrecisionAndMissing(t *testing.T) {
	c := cache.NewRedis(newClient(t))
	ctx := t.Context()
	require.ErrorIs(t, c.Expire(ctx, "missing", time.Second), cache.ErrNotExist)
	require.NoError(t, c.Store(ctx, "short", []byte("x"), 1500*time.Millisecond))
	ttl, err := c.TTL(ctx, "short")
	require.NoError(t, err)
	require.Greater(t, ttl, time.Second)
}
