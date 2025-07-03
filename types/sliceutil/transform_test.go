package sliceutil_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		n2x := sliceutil.Map(n, func(val int) int {
			return val * 2
		})
		assert.Equal([]int{2, 4, 6, 8, 10}, n2x)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		n2x := sliceutil.Map(n, func(val int) int {
			return val * 2
		})
		assert.Equal([]int{}, n2x)
	})
}

func TestMapError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 2, 3, 4, 5}
		n2x, err := sliceutil.MapError(n, func(val int) (int, error) {
			return val * 2, nil
		})
		assert.Equal([]int{2, 4, 6, 8, 10}, n2x)
		assert.Nil(err)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		n2x, err := sliceutil.MapError(n, func(val int) (int, error) {
			return val * 2, nil
		})
		assert.Equal([]int{}, n2x)
		assert.Nil(err)
	})

	t.Run("failed", func(t *testing.T) {
		assert := assert.New(t)

		wantErr := errors.New("test: want error")

		n := []int{1, 2, 3, 4, 5}
		n2x, err := sliceutil.MapError(n, func(val int) (int, error) {
			if val == 3 {
				return 0, wantErr
			}
			return val * 2, nil
		})
		assert.Equal([]int(nil), n2x)
		assert.ErrorIs(err, wantErr)
	})
}

func TestDedup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{1, 1, 1, 1, 1}
		unique := sliceutil.Dedup(n)
		assert.ElementsMatch([]int{1}, unique)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []int{}
		unique := sliceutil.Dedup(n)
		assert.Equal([]int{}, unique)
	})
}

func TestDedupFunc(t *testing.T) {
	type user struct {
		Age int
	}

	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		n := []user{{30}, {20}, {30}, {10}}
		unique := sliceutil.DedupFunc(n, func(u user) int {
			return u.Age
		})
		assert.ElementsMatch([]user{{30}, {20}, {10}}, unique)
	})

	t.Run("empty", func(t *testing.T) {
		assert := assert.New(t)

		n := []user{}
		unique := sliceutil.DedupFunc(n, func(u user) int {
			return u.Age
		})
		assert.Equal([]user{}, unique)
	})
}
