package batch_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/batch"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestCache(t *testing.T) {
	cache := batch.NewCache[int, int]()

	// Load non-existing keys should return no values.
	res, err := cache.LoadMany(ctx, 1, 2, 3)
	is := assert.New(t)
	is.Nil(err)
	is.Len(res, 0)

	kv := map[int]int{
		1: 100,
		2: 200,
		3: 300,
	}

	// Store the keys with ttl of 10ms.
	err = cache.StoreMany(ctx, kv, 10*time.Millisecond)
	is.Nil(err)

	// Load the keys should return the values.
	// Non-existing keys should return no values.
	res, err = cache.LoadMany(ctx, 1, 2, 3, 99)
	is.Nil(err)
	is.Len(res, 3)
	is.Equal(res, kv)

	// Wait for cache to expire.
	time.Sleep(10 * time.Millisecond)
	res, err = cache.LoadMany(ctx, 1, 2, 3, 99)
	is.Nil(err)
	is.Len(res, 0)
}
