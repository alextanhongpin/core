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
		key := t.Name()

		is := assert.New(t)
		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)
	})

	t.Run("add once", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name()

		is := assert.New(t)
		ok, err := cf.AddNX(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.AddNX(ctx, key, "foo")
		is.Nil(err)
		is.False(ok)
	})

	t.Run("count", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name()

		is := assert.New(t)
		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		n, err := cf.Count(ctx, key, "foo")
		is.Nil(err)
		is.Equal(int64(2), n, "foo is added twice")
	})

	t.Run("delete", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name()

		is := assert.New(t)
		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Delete(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Delete(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		ok, err = cf.Delete(ctx, key, "foo")
		is.Nil(err)
		is.False(ok, "fully deleted")
	})

	t.Run("exists", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name()

		is := assert.New(t)
		exists, err := cf.Exists(ctx, key, "foo")
		is.Nil(err)
		is.False(exists)

		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		exists, err = cf.Exists(ctx, key, "foo")
		is.Nil(err)
		is.True(exists)
	})

	t.Run("mexists", func(t *testing.T) {
		cf := probs.NewCuckooFilter(redistest.Client(t))
		key := t.Name()

		is := assert.New(t)
		ok, err := cf.Add(ctx, key, "foo")
		is.Nil(err)
		is.True(ok)

		exists, err := cf.MExists(ctx, key, "foo", "bar")
		is.Nil(err)
		is.True(exists[0])
		is.False(exists[1])
	})
}
