package jsonl_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/types/jsonl"
	"github.com/go-openapi/testify/assert"
)

func TestJSONL(t *testing.T) {
	name := fmt.Sprintf("testdata/%s.jsonl", t.Name())
	jsonl := jsonl.New[int](name)
	err := jsonl.Remove()
	is := assert.New(t)
	is.NoError(err)

	err = jsonl.Store(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	is.NoError(err)

	res, err := jsonl.Load()
	is.NoError(err)
	is.Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, res)

	res, err = jsonl.Tail(1)
	is.NoError(err)
	is.Equal([]int{10}, res)

	res, err = jsonl.Tail(3)
	is.NoError(err)
	is.Equal([]int{8, 9, 10}, res)

	res, err = jsonl.Tail(5)
	is.NoError(err)
	is.Equal([]int{6, 7, 8, 9, 10}, res)

	res, err = jsonl.Tail(20)
	is.NoError(err)
	is.Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, res)

	res, err = jsonl.Head(5)
	is.NoError(err)
	is.Equal([]int{1, 2, 3, 4, 5}, res)
}
