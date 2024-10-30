package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestCountMinSketch(t *testing.T) {
	cms := probs.NewCountMinSketch(redistest.Client(t))
	key := t.Name() + ":cms:events"

	t.Run("init", func(t *testing.T) {})
	t.Run("incr by", func(t *testing.T) {

	})

	t.Run("incr by - key does not exist", func(t *testing.T) {
		_, err := cms.IncrBy(ctx, t.Name(), map[any]int{
			"hello": 10,
		})
		is := assert.New(t)
		is.Nil(err)
	})
	t.Run("merge", func(t *testing.T) {})
	t.Run("merge with weight", func(t *testing.T) {})
	t.Run("query", func(t *testing.T) {
		cms.IncrBy(ctx, key, map[any]int{
			"key": 1,
		})
	})
}
