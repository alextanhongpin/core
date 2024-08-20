package batch_test

import (
	"testing"

	"github.com/alextanhongpin/core/sync/batch"
	"github.com/stretchr/testify/assert"
)

func TestQueue_Load(t *testing.T) {
	loader := newBatchLoader()
	q := batch.NewQueue(loader)

	q.Add(1)
	q.Flush(ctx)

	v, err := q.Load(1)
	is := assert.New(t)
	is.Nil(err)
	is.Equal("1", v)

	v, err = q.Load(-1)
	is.ErrorIs(err, batch.ErrKeyNotExist)
	is.Equal("", v)
}

func TestQueue_LoadMany(t *testing.T) {
	loader := newBatchLoader()
	q := batch.NewQueue(loader)
	q.Add(1, 2, 3)
	q.Flush(ctx)

	vs, err := q.LoadMany([]int{1, 2, 3})
	is := assert.New(t)
	is.Nil(err)
	is.Equal([]string{"1", "2", "3"}, vs)

	vs, err = q.LoadMany([]int{-1, -2, -3})
	is.Nil(err, err)
	is.Empty(vs)
}
