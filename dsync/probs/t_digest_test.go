package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestTDigest(t *testing.T) {
	key := t.Name()

	t.Run("add", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		status, err := td.Add(ctx, key, 10, 20, 30)
		is.Nil(err)
		is.Equal("OK", status)
	})

	t.Run("cdf", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		cdf, err := td.CDF(ctx, key, 10, 20, 30)
		is.Nil(err)
		is.GreaterOrEqual(0.16666666666666666, cdf[0])
		is.Equal(0.5, cdf[1])
		is.GreaterOrEqual(0.8333333333333334, cdf[2])
	})
	// TODO: Test quantile

	t.Run("min", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		f, err := td.Min(ctx, key)
		is.Nil(err)
		is.Equal(float64(10), f)
	})

	t.Run("max", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		f, err := td.Max(ctx, key)
		is.Nil(err)
		is.Equal(float64(30), f)
	})

	t.Run("rank", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		ranks, err := td.Rank(ctx, key, 10, 30)
		is.Nil(err)
		is.Equal([]int64{0, 2}, ranks)
	})

	t.Run("rev rank", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		ranks, err := td.RevRank(ctx, key, 10, 30)
		is.Nil(err)
		is.Equal([]int64{2, 0}, ranks)
	})

	t.Run("by rank", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		ranks, err := td.ByRank(ctx, key, 0, 2)
		is.Nil(err)
		is.Equal([]float64{10, 30}, ranks)
	})

	t.Run("by rev rank", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		ranks, err := td.ByRevRank(ctx, key, 0, 2)
		is.Nil(err)
		is.Equal([]float64{30, 10}, ranks)
	})

	t.Run("trimmed mean", func(t *testing.T) {
		td := probs.NewTDigest(redistest.Client(t))

		is := assert.New(t)
		mean, err := td.TrimmedMean(ctx, key, 0.1, 0.9)
		is.Nil(err)
		is.Equal(float64(20), mean)
	})
}
