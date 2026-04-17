package jsonl_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/alextanhongpin/core/types/jsonl"
	"github.com/go-openapi/testify/assert"
)

func TestJSONL(t *testing.T) {
	name := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	f, err := jsonl.OpenFile[int](name, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, f.Close())
	})

	t.Run("read empty", func(t *testing.T) {
		seq, stop := f.ReadLines()
		lines := slices.Collect(seq)

		is := assert.New(t)
		is.NoError(stop())
		is.Empty(lines)
	})

	t.Run("write and read", func(t *testing.T) {
		is := assert.New(t)

		var res []int
		for i := range 10 {
			res = append(res, i)
			err = f.Write(i)
			is.NoError(err)

			seq, stop := f.ReadLines()
			lines := slices.Collect(seq)
			is.NoError(stop())
			is.Equal(res, lines)
		}
	})

	t.Run("copy", func(t *testing.T) {
		oldFile := f
		newFile, err := jsonl.OpenFile[int](fmt.Sprintf("testdata/%s.jsonl", t.Name()), os.O_RDWR|os.O_CREATE|os.O_TRUNC)

		is := assert.New(t)
		is.NoError(err)

		err = jsonl.Copy(oldFile, newFile)
		is.NoError(err)

		seq, stop := newFile.ReadLines()
		lines := slices.Collect(seq)
		is.NoError(stop())
		is.Equal([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, lines)
	})

	t.Run("copy func", func(t *testing.T) {
		oldFile := f
		newFile, err := jsonl.OpenFile[int](fmt.Sprintf("testdata/%s.jsonl", t.Name()), os.O_RDWR|os.O_CREATE|os.O_TRUNC)

		is := assert.New(t)
		is.NoError(err)

		err = jsonl.CopyFunc(oldFile, newFile, func(i int) bool {
			return i%2 == 0
		})
		is.NoError(err)

		seq, stop := newFile.ReadLines()
		lines := slices.Collect(seq)
		is.NoError(stop())
		is.Equal([]int{0, 2, 4, 6, 8}, lines)
	})
}
