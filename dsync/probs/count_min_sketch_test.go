package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestCountMinSketch(t *testing.T) {

	t.Run("init twice", func(t *testing.T) {
		key := t.Name()

		cms := probs.NewCountMinSketch(redistest.Client(t))
		_, err := cms.Init(ctx, key)
		is := assert.New(t)
		is.Nil(err)

		_, err = cms.Init(ctx, key)
		is.Nil(err)
	})

	t.Run("incr by", func(t *testing.T) {
		cms := probs.NewCountMinSketch(redistest.Client(t))
		counts, err := cms.IncrBy(ctx, t.Name(), map[string]int64{
			"bar": 2,
			"foo": 1,
		})
		is := assert.New(t)
		is.Nil(err)
		is.Equal([]int64{2, 1}, counts)
	})
	t.Run("merge", func(t *testing.T) {
		cms := probs.NewCountMinSketch(redistest.Client(t))
		key1 := t.Name() + ":1"
		key2 := t.Name() + ":2"
		key3 := t.Name() + ":3"

		_, err := cms.IncrBy(ctx, key1, map[string]int64{
			"bar": 2,
			"foo": 1,
		})
		is := assert.New(t)
		is.Nil(err)

		_, err = cms.IncrBy(ctx, key2, map[string]int64{
			"bar": 1,
			"foo": 2,
		})
		is.Nil(err)

		status, err := cms.Merge(ctx, key3, key2, key1)
		is.Nil(err)
		is.Equal("OK", status)

		counts, err := cms.Query(ctx, key3, "foo", "bar")
		is.Nil(err)
		is.Equal([]int64{3, 3}, counts)
	})

	t.Run("merge with weight", func(t *testing.T) {
		cms := probs.NewCountMinSketch(redistest.Client(t))
		key1 := t.Name() + ":1"
		key2 := t.Name() + ":2"
		key3 := t.Name() + ":3"

		_, err := cms.IncrBy(ctx, key1, map[string]int64{
			"foo": 1,
			"bar": 2,
		})
		is := assert.New(t)
		is.Nil(err)

		_, err = cms.IncrBy(ctx, key2, map[string]int64{
			"foo": 2,
			"bar": 1,
		})
		is.Nil(err)

		status, err := cms.MergeWithWeight(ctx, key3, map[string]int64{
			key1: 2,
			key2: 4,
		})
		is.Nil(err)
		is.Equal("OK", status)

		counts, err := cms.Query(ctx, key3, "foo", "bar")
		is.Nil(err)
		is.Equal([]int64{10, 8}, counts)
	})

	t.Run("query", func(t *testing.T) {
		cms := probs.NewCountMinSketch(redistest.Client(t))
		_, err := cms.IncrBy(ctx, t.Name(), map[string]int64{
			"foo": 2,
			"bar": 1,
		})
		is := assert.New(t)
		is.Nil(err)

		counts, err := cms.Query(ctx, t.Name(), "foo", "bar", "baz")
		is.Nil(err)
		is.Equal([]int64{2, 1, 0}, counts)
	})
}
