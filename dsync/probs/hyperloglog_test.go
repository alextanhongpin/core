package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestHyperLogLog(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		hll := probs.NewHyperLogLog(redistest.Client(t))
		key := t.Name() + ":hll:page_views"
		is := assert.New(t)
		n, err := hll.Add(ctx, key, "a", "a", 1, 1, true, false)
		is.Nil(err)
		is.Equal(int64(1), n)
		n, err = hll.Count(ctx, key)
		is.Nil(err)
		is.Equal(int64(3), n)
	})

	t.Run("merge", func(t *testing.T) {
		hll := probs.NewHyperLogLog(redistest.Client(t))

		today := t.Name() + ":hll:page_views:today"
		yesterday := t.Name() + ":hll:page_views:yesterday"
		alltime := t.Name() + ":hll:page_views:alltime"

		is := assert.New(t)
		n, err := hll.Add(ctx, today, "a", "b", "c")
		is.Nil(err)
		is.Equal(int64(1), n)

		n, err = hll.Add(ctx, yesterday, "b", "c", "e")
		is.Nil(err)
		is.Equal(int64(1), n)

		_, err = hll.Merge(ctx, alltime, today, yesterday)
		is.Nil(err)

		n, err = hll.Count(ctx, alltime)
		is.Nil(err)
		is.Equal(int64(4), n)
	})
}
