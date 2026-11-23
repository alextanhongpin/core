package fstring_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/types/fstring"
	"github.com/go-openapi/testify/assert"
)

func TestFString(t *testing.T) {
	t.Run("any value", func(t *testing.T) {
		f, err := fstring.ReadFile[any]("testdata/system.txt")
		is := assert.New(t)
		is.NoError(err)

		s, err := f.Format("world!")
		is.NoError(err)
		is.Equal("hello world!\n", s)
	})

	t.Run("any nil", func(t *testing.T) {
		f, err := fstring.ReadFile[any]("testdata/system.txt")
		is := assert.New(t)
		is.NoError(err)

		s, err := f.Format(nil)
		is.NoError(err)
		is.Equal("hello <no value>\n", s)
	})

	t.Run("typed empty", func(t *testing.T) {
		f, err := fstring.ReadFile[string]("testdata/system.txt")
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

// Regression: invalid template syntax used to panic.
func TestFStringParseError(t *testing.T) {
	_, err := fstring.FString[any]("{{ .Foo").Format(nil)
	assert.Error(t, err)
}

func TestFStringExecError(t *testing.T) {
	type data struct{ Name string }
	s, err := fstring.FString[data]("hi {{ .Missing }}").Format(data{"x"})
	assert.Error(t, err)
	assert.Empty(t, s)
}

func TestFStringParseJSON(t *testing.T) {
	s, err := fstring.FString[string](`{{ (parse_json .).name }}`).FormatFunc(`{"name":"alice"}`, fstring.FuncMap)
	assert.NoError(t, err)
	assert.Equal(t, "alice", s)
}

func TestReadFileMissing(t *testing.T) {
	_, err := fstring.ReadFile[any]("testdata/missing.txt")
	assert.Error(t, err)
}

func ExampleFString_Format() {
	type user struct{ Name string }
	s, err := fstring.FString[user]("Hello {{ .Name }}!").Format(user{"Alice"})
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	// Output: Hello Alice!
}
