package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestBloomFilter(t *testing.T) {
	bf := probs.NewBloomFilter(redistest.Client(t))
	key := t.Name() + ":bf:users"

	t.Run("add", func(t *testing.T) {
		is := assert.New(t)
		ok, err := bf.Add(ctx, key, t.Name())
		is.Nil(err)
		is.True(ok)
	})

	t.Run("add multiple", func(t *testing.T) {
		is := assert.New(t)
		ok, err := bf.Add(ctx, key, t.Name())
		is.Nil(err)
		is.True(ok)

		ok, err = bf.Add(ctx, key, t.Name())
		is.Nil(err)
		is.False(ok, "can only add once")
	})

	t.Run("madd", func(t *testing.T) {
		is := assert.New(t)
		oks, err := bf.MAdd(ctx, key, t.Name(), "foo", 42, true, "foo")
		is.Nil(err)
		is.True(oks[0])
		is.True(oks[1])
		is.True(oks[2])
		is.True(oks[3], "already added before")
	})

	t.Run("exists", func(t *testing.T) {
		is := assert.New(t)
		ok, err := bf.Add(ctx, key, t.Name())
		is.Nil(err)
		is.True(ok)

		ok, err = bf.Exists(ctx, key, t.Name())
		is.Nil(err)
		is.True(ok)

		ok, err = bf.Exists(ctx, key, "unknown")
		is.Nil(err)
		is.False(ok)
	})

	t.Run("mexists", func(t *testing.T) {
		is := assert.New(t)
		ok, err := bf.Add(ctx, key, t.Name())
		is.Nil(err)
		is.True(ok)

		oks, err := bf.MExists(ctx, key, t.Name(), "unknown")
		is.Nil(err)
		is.True(oks[0])
		is.False(oks[1])
	})
}
