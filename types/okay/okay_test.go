package okay_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/okay"
	"github.com/stretchr/testify/assert"
)

func TestZero(t *testing.T) {
	ok := okay.New[any]()

	ctx := context.Background()
	valid, err := ok.All(ctx, "hello").Unwrap()

	assert := assert.New(t)
	assert.False(valid)
	assert.ErrorIs(err, okay.Invalid)
}

func TestOkay(t *testing.T) {
	checkDir := func(_ context.Context, file string) okay.Response {
		if strings.HasPrefix(file, "/Home/Alice") {
			return okay.Allow(true)
		}

		return okay.Errorf("not a valid path")
	}

	checkFileType := func(_ context.Context, file string) okay.Response {
		if ext := filepath.Ext(file); ext != ".txt" {
			return okay.Errorf("cannot read non-text files")
		}

		return okay.Allow(true)
	}

	ctx := context.Background()

	ok := okay.New[string]()
	ok.Add(checkDir)
	ok.Add(checkFileType)

	t.Run("all success", func(t *testing.T) {
		valid, err := ok.All(ctx, "/Home/Alice/todo.txt").Unwrap()

		assert := assert.New(t)
		assert.True(valid)
		assert.Nil(err)
	})

	t.Run("all failed", func(t *testing.T) {
		valid, err := ok.All(ctx, "/Home/Alice/todo.json").Unwrap()

		assert := assert.New(t)
		assert.False(valid)
		assert.NotNil(err)
		assert.Equal("cannot read non-text files", err.Error())
	})

	t.Run("any all", func(t *testing.T) {
		valid, err := ok.Any(ctx, "/Home/Alice/todo.txt").Unwrap()

		assert := assert.New(t)
		assert.True(valid)
		assert.Nil(err)
	})

	t.Run("any partial", func(t *testing.T) {
		valid, err := ok.Any(ctx, "/Home/Alice/todo.json").Unwrap()

		assert := assert.New(t)
		assert.True(valid)
		assert.Nil(err)
	})

	t.Run("none all", func(t *testing.T) {
		valid, err := ok.None(ctx, "/Home/Alice/todo.txt").Unwrap()

		assert := assert.New(t)
		assert.False(valid)
		assert.ErrorIs(err, okay.Denied)
	})

	t.Run("none some", func(t *testing.T) {
		valid, err := ok.None(ctx, "/Home/Alice/todo.json").Unwrap()

		assert := assert.New(t)
		assert.False(valid)
		assert.ErrorIs(err, okay.Denied)
	})

	t.Run("none none", func(t *testing.T) {
		valid, err := ok.None(ctx, "/Home/Bob/todo.json").Unwrap()

		assert := assert.New(t)
		assert.True(valid)
		assert.Nil(err)
	})
}
