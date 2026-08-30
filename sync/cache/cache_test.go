package cache_test

import (
	"runtime"
	"testing"

	"github.com/alextanhongpin/core/sync/cache"
)

func TestCache(t *testing.T) {
	c := cache.New(func(string) (*int, error) {
		return new(42), nil
	})

	key := t.Name()
	t.Run("create", func(t *testing.T) {
		v, loaded, err := c.LoadOrCreate(key)
		if err != nil {
			t.Fatal(err)
		}
		if loaded {
			t.Fatalf("want not loaded, got true")
		}
		if *v != 42 {
			t.Fatalf("want 42, got %v", v)
		}
	})

	t.Run("loaded", func(t *testing.T) {
		v, loaded, err := c.LoadOrCreate(key)
		if err != nil {
			t.Fatal(err)
		}
		if !loaded {
			t.Fatalf("want loaded, got false")
		}
		if *v != 42 {
			t.Fatalf("want 42, got %v", v)
		}
	})

	t.Run("gc", func(t *testing.T) {
		// This will clear the cache.
		runtime.GC()

		v, loaded, err := c.LoadOrCreate(key)
		if err != nil {
			t.Fatal(err)
		}
		if loaded {
			t.Fatalf("want not loaded, got true")
		}
		if *v != 42 {
			t.Fatalf("want 42, got %v", v)
		}
	})
}
