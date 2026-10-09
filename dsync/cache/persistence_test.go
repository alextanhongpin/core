package cache_test

import (
	"errors"
	"github.com/alextanhongpin/core/dsync/cache"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileOwnsValuesAndPersistsExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	f, err := cache.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte("original")
	if err := f.Store(t.Context(), "k", value, 0); err != nil {
		t.Fatal(err)
	}
	value[0] = 'x'
	got, err := f.Load(t.Context(), "k")
	if err != nil || string(got) != "original" {
		t.Fatalf("%q %v", got, err)
	}
	got[0] = 'x'
	if ttl, err := f.TTL(t.Context(), "k"); err != nil || ttl != -1 {
		t.Fatalf("TTL %v %v", ttl, err)
	}
	if err := f.Expire(t.Context(), "k", time.Hour); err != nil {
		t.Fatal(err)
	}
	f.Close()
	reopened, err := cache.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if ttl, err := reopened.TTL(t.Context(), "k"); err != nil || ttl <= 0 {
		t.Fatalf("expiry lost: %v %v", ttl, err)
	}
	if err := f.Store(t.Context(), "k", []byte("closed"), 0); !errors.Is(err, cache.ErrClosed) {
		t.Fatal(err)
	}
}
func TestFSExpiryResetDeleteAndSize(t *testing.T) {
	dir := t.TempDir()
	f, err := cache.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, key := range []string{"a", "b"} {
		if err := f.Store(t.Context(), key, []byte(key), time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Store(t.Context(), "a", []byte("persistent"), 0); err != nil {
		t.Fatal(err)
	}
	if ttl, err := f.TTL(t.Context(), "a"); err != nil || ttl != -1 {
		t.Fatalf("stale TTL: %v %v", ttl, err)
	}
	if err := f.Expire(t.Context(), "b", time.Hour); err != nil {
		t.Fatal(err)
	}
	if n, err := f.Size(t.Context()); err != nil || n != 2 {
		t.Fatalf("size %d %v", n, err)
	}
	if n, err := f.Delete(t.Context(), "a", "b", "missing"); err != nil || n != 2 {
		t.Fatalf("delete %d %v", n, err)
	}
}
func TestFSExpiredEntryIsRemoved(t *testing.T) {
	f, err := cache.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Store(t.Context(), "k", []byte("v"), time.Nanosecond)
	if _, err := f.Load(t.Context(), "k"); !errors.Is(err, cache.ErrNotExist) {
		t.Fatal(err)
	}
	if n, err := f.Size(t.Context()); err != nil || n != 0 {
		t.Fatalf("expired file retained: %d %v", n, err)
	}
}
func TestFileDecodeFailureReleasesResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("invalid"), 0600)
	if _, err := cache.NewFile(path); err == nil {
		t.Fatal("bad JSON accepted")
	}
	os.WriteFile(path, []byte("{}"), 0600)
	f, err := cache.NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
