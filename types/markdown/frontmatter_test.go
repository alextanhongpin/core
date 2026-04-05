package markdown_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/markdown"
	"github.com/go-openapi/testify/assert"
)

func TestWriteFrontmatter(t *testing.T) {
	meta := map[string]any{
		"name":        t.Name(),
		"description": t.Name(),
	}
	var bb bytes.Buffer
	w := io.MultiWriter(&bb, t.Output())
	err := markdown.WriteFrontmatter(w, meta)
	is := assert.New(t)
	is.NoError(err)

	want := `---
description: TestWriteFrontmatter
name: TestWriteFrontmatter
---
`
	is.Equal(want, bb.String())
}

func TestParseFrontmatter(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := `---
name: a name
description: a description
---
hello world`
		meta, r, err := markdown.ParseFrontmatter(strings.NewReader(s))
		is := assert.New(t)
		is.NoError(err)

		b, err := io.ReadAll(r)
		is.NoError(err)
		is.Equal("hello world", string(b))
		is.Equal(map[string]any{"name": "a name", "description": "a description"}, meta)
	})

	t.Run("no frontmatter", func(t *testing.T) {
		meta, r, err := markdown.ParseFrontmatter(strings.NewReader(t.Name()))
		is := assert.New(t)
		is.NoError(err)

		b, err := io.ReadAll(r)
		is.NoError(err)
		is.Equal(t.Name(), string(b))
		is.Nil(meta)
	})

	t.Run("empty", func(t *testing.T) {
		meta, r, err := markdown.ParseFrontmatter(strings.NewReader(""))
		is := assert.New(t)
		is.ErrorIs(err, io.EOF)
		is.Nil(r)
		is.Nil(meta)
	})
}
