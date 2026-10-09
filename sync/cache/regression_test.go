package cache_test

import (
	"github.com/alextanhongpin/core/sync/cache"
	"testing"
)

func TestNilValue(t *testing.T) {
	c := cache.New(func(string) (*int, error) { return nil, nil })
	if _, _, err := c.LoadOrCreate("nil"); err == nil {
		t.Fatal("nil creation must return an error")
	}
}
