package batch_test

import (
	"strconv"
	"testing"

	"github.com/alextanhongpin/core/sync/batch"
	"github.com/stretchr/testify/assert"
)

func TestLoader_Load(t *testing.T) {
	loader := newBatchLoader()
	t.Run("exist", func(t *testing.T) {
		v, err := loader.Load(ctx, 1)
		is := assert.New(t)
		is.Nil(err)
		is.Equal("1", v)
	})

	t.Run("not exist", func(t *testing.T) {
		v, err := loader.Load(ctx, -99)
		is := assert.New(t)
		is.ErrorIs(err, batch.ErrKeyNotExist)
		is.Equal("", v)
	})
}

func TestLoader_LoadMany(t *testing.T) {
	loader := newBatchLoader()
	t.Run("exist", func(t *testing.T) {
		vs, err := loader.LoadMany(ctx, []int{1, 2, 3})
		is := assert.New(t)
		is.Nil(err)
		is.Equal([]string{"1", "2", "3"}, vs)
	})

	t.Run("not exist", func(t *testing.T) {
		vs, err := loader.LoadMany(ctx, []int{-99, -100, -101})
		is := assert.New(t)
		is.Nil(err)
		is.Len(vs, 0)
	})
}

func TestLoader_LoadManyResult(t *testing.T) {
	loader := newBatchLoader()
	t.Run("exist", func(t *testing.T) {
		rs, err := loader.LoadManyResult(ctx, []int{1, 2, 3})
		is := assert.New(t)
		is.Nil(err)

		tc := []struct {
			key int
			val string
		}{
			{1, "1"},
			{2, "2"},
			{3, "3"},
		}
		for _, c := range tc {
			v, err := rs[c.key].Unwrap()
			is.Nil(err)
			is.Equal(c.val, v)
		}
	})

	t.Run("not exist", func(t *testing.T) {
		rs, err := loader.LoadManyResult(ctx, []int{-99, -100, -101})
		is := assert.New(t)
		is.Nil(err)
		for _, r := range rs {
			v, err := r.Unwrap()
			is.ErrorIs(err, batch.ErrKeyNotExist)
			is.Empty(v)
		}
	})
}

func newBatchLoader() *batch.Loader[int, string] {
	return batch.NewLoader(&batch.LoaderOptions[int, string]{
		BatchFn: func(ks []int) (map[int]string, error) {
			res := make(map[int]string)
			for _, k := range ks {
				// Only positive number is allowed.
				if k <= 0 {
					continue
				}
				res[k] = strconv.Itoa(k)
			}

			return res, nil
		},
	})
}
