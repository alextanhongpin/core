package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestTopK(t *testing.T) {
	client := redistest.Client(t)
	// Find top k hashtag
	topK := probs.NewTopK(client)
	key := t.Name() + ":top_k:hashtag"

	is := assert.New(t)
	status, err := topK.Create(ctx, key, 5)
	is.Nil(err)
	is.Equal("OK", status)
	_, err = topK.Add(ctx, key,
		"ai",
		"ml", "ml",
		"js", "js",
		"python", "python",
		"ts", "ts", "ts",
		"go", "go", "go", "go", "go", "go",
	)
	is.Nil(err)

	t.Run("count", func(t *testing.T) {
		counts, err := topK.Count(ctx, key, "go", "ts", "ai")

		is := assert.New(t)
		is.Nil(err)
		is.Equal(counts, []int64{6, 3, 1})
	})

	t.Run("incr by", func(t *testing.T) {
		evt := new(probs.Event)
		evt.Add("go", 10)
		evt.Add("ts", 10)

		_, err := topK.IncrBy(ctx, key, evt)
		is := assert.New(t)
		is.Nil(err)
	})

	t.Run("list", func(t *testing.T) {
		list, err := topK.List(ctx, key)
		is := assert.New(t)
		is.Nil(err)
		is.Equal(list, []string{"go", "ts", "js", "ml", "python"})
	})

	t.Run("list with count", func(t *testing.T) {
		listWithCount, err := topK.ListWithCount(ctx, key)
		is := assert.New(t)
		is.Nil(err)
		is.Equal(listWithCount, map[string]int64{
			"go":     16,
			"ts":     13,
			"python": 2,
			"js":     2,
			"ml":     2,
		})
	})

	t.Run("query", func(t *testing.T) {
		list, err := topK.Query(ctx, key, "go", "unknown")
		is := assert.New(t)
		is.Nil(err)
		is.Equal(list, []bool{true, false})
	})
}
