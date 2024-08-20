package batch_test

import (
	"testing"

	"github.com/alextanhongpin/core/sync/batch"
	"github.com/stretchr/testify/assert"
)

func TestWaitGroup_Load(t *testing.T) {
	loader := newBatchLoader()
	bwg := batch.NewWaitGroup(loader)
	bwg.Add(2)

	is := assert.New(t)

	go func() {
		v, err := bwg.Load(1)
		is.Nil(err)
		is.Equal("1", v)
	}()

	go func() {
		v, err := bwg.Load(-1)
		is.ErrorIs(err, batch.ErrKeyNotExist)
		is.Equal("", v)
	}()

	is.Nil(bwg.Wait(ctx))
}

func TestWaitGroup_LoadMany(t *testing.T) {
	loader := newBatchLoader()
	bwg := batch.NewWaitGroup(loader)
	bwg.Add(2)

	is := assert.New(t)

	go func() {
		vs, err := bwg.LoadMany([]int{1, 2, 3})
		is.Nil(err)
		is.Equal([]string{"1", "2", "3"}, vs)
	}()

	go func() {
		vs, err := bwg.LoadMany([]int{-1, -2, -3})
		is.ErrorIs(err, batch.ErrKeyNotExist)
		is.Empty(vs)
	}()

	is.Nil(bwg.Wait(ctx))
}
