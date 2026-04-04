package fstring_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/fstring"
	"github.com/go-openapi/testify/assert"
)

func TestFString(t *testing.T) {
	t.Run("any value", func(t *testing.T) {
		f, err := fstring.FStringFromFile[any]("testdata/system.txt")
		is := assert.New(t)
		is.NoError(err)

		s, err := f.Format("world!")
		is.NoError(err)
		is.Equal("hello world!\n", s)
	})

	t.Run("any nil", func(t *testing.T) {
		f, err := fstring.FStringFromFile[any]("testdata/system.txt")
		is := assert.New(t)
		is.NoError(err)

		s, err := f.Format(nil)
		is.NoError(err)
		is.Equal("hello <no value>\n", s)
	})

	t.Run("typed empty", func(t *testing.T) {
		f, err := fstring.FStringFromFile[string]("testdata/system.txt")
		is := assert.New(t)
		is.NoError(err)

		s, err := f.Format("")
		is.NoError(err)
		is.Equal("hello \n", s)
	})

	t.Run("format json", func(t *testing.T) {
		f := fstring.FString[any]("{{ json . }}")
		is := assert.New(t)
		s, err := f.FormatFunc(map[string]any{
			"foo": "bar",
		}, fstring.FuncMap)
		is.NoError(err)
		is.Equal(`{"foo":"bar"}`, s)
	})
}
