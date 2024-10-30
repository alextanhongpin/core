package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestCuckooFilter(t *testing.T) {
	t.Run("add multiple times", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name() + ":cf:requests_total"

		is := assert.New(t)
		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

	})

	t.Run("add once", func(t *testing.T) {
	})

	t.Run("mexists", func(t *testing.T) {})
	t.Run("exists", func(t *testing.T) {})
	t.Run("count", func(t *testing.T) {})
	t.Run("delete", func(t *testing.T) {})
}
