package cache_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// FS must replace value files atomically: a reader holding the old file sees
// the old bytes, never a truncated or partially rewritten file.
func TestFSReplacesFilesAtomically(t *testing.T) {
	dir := t.TempDir()
	fs, err := cache.NewFS(dir)
	require.NoError(t, err)
	defer fs.Close()
	ctx := t.Context()
	require.NoError(t, fs.Store(ctx, "k", []byte("old"), 0))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var name string
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			name = e.Name()
		}
	}
	require.NotEmpty(t, name)
	old, err := os.Open(filepath.Join(dir, name))
	require.NoError(t, err)
	defer old.Close()

	require.NoError(t, fs.Store(ctx, "k", []byte("new and longer"), time.Hour))
	b, err := io.ReadAll(old)
	require.NoError(t, err)
	require.Equal(t, "old", string(b))

	v, err := fs.Load(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "new and longer", string(v))
	n, err := fs.Size(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

// DEL with no keys is a Redis error; Delete() must report zero like File and FS.
func TestRedisDeleteNoKeys(t *testing.T) {
	n, err := cache.MustNewRedis(newClient(t)).Delete(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
}
