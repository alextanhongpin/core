package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestHyperLogLog(t *testing.T) {
	hll := probs.NewHyperLogLog(redistest.Client(t))
	// Use hyperloglog to track page views.
	key := t.Name() + ":hll:page_views"

	is := assert.New(t)
	n, err := hll.Add(ctx, key, "a", "a", 1, 1, true, false)
	is.Nil(err)
	is.Equal(int64(4), n)

	n, err = hll.Count(ctx, key)
	is.Nil(err)
	is.Equal(int64(4), n)
}
