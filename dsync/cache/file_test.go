package cache_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/assert"
)

func TestFileStorage(t *testing.T) {
	path := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(path))
	})

	c, err := cache.NewFile(path)
	assert.NoError(t, err)

	defer c.Close()
	testStorage(t, c)
}

func TestFileStorageInit(t *testing.T) {
	is := assert.New(t)
	path := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	t.Cleanup(func() {
		is.NoError(os.Remove(path))
	})

	c, err := cache.NewFile(path)
	is.NoError(err)
	is.NoError(c.Store(t.Context(), t.Name(), []byte(t.Name()), -time.Second))
	n, err := c.Size(t.Context())
	is.Equal(1, n)
	is.NoError(err)
	c.Close()

	b, err := os.ReadFile(path)
	is.NoError(err)
	is.NotEmpty(b)

	c, err = cache.NewFile(path)
	is.NoError(err)
	defer c.Close()

	n, err = c.Size(t.Context())
	is.NoError(err)
	is.Equal(0, n)
}
