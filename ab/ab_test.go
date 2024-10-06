package ab_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/ab"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	_ = m.Run()
}

func TestUnique(t *testing.T) {
	unique := ab.NewUnique(redistest.Client(t))
	added, err := unique.Store(ctx, "key", "val")
	is := assert.New(t)
	is.Nil(err)
	is.True(added)

	added, err = unique.Store(ctx, "key", "val")
	is.Nil(err)
	is.False(added)

	count, err := unique.Load(ctx, "key")
	is.Nil(err)
	is.Equal(int64(1), count)
}
