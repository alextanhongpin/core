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
	var bb bytes.Buffer
	err := markdown.WriteFrontmatter(&bb, map[string]any{"name": "alice", "description": "d"})
	assert.NoError(t, err)
	assert.Equal(t, "---\ndescription: d\nname: alice\n---\n", bb.String())
}

func TestParseFrontmatter(t *testing.T) {
	type meta struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name, in, body string
		want           meta
		err            error
	}{
		{name: "success", in: "---\nname: a\n---\nhello world", want: meta{"a"}, body: "hello world"},
		{name: "crlf", in: "---\r\nname: a\r\n---\r\nbody", want: meta{"a"}, body: "body"},
		{name: "closing delimiter at eof", in: "---\nname: a\n---", want: meta{"a"}},
		{name: "empty frontmatter", in: "---\n---\nbody", body: "body"},
		{name: "no frontmatter", in: "hello\nworld", body: "hello\nworld"},
		// Regression: input shorter than the delimiter used to return io.EOF.
		{name: "short", in: "hi", body: "hi"},
		{name: "empty", in: ""},
		{name: "delimiter only", in: "---", body: "---"},
		{name: "unterminated", in: "---\nname: a\n", err: markdown.ErrUnterminated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, r, err := markdown.ParseFrontmatter[meta](strings.NewReader(tt.in))
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
			b, err := io.ReadAll(r)
			assert.NoError(t, err)
			assert.Equal(t, tt.body, string(b))
		})
	}
}

func TestParseFrontmatterInvalidYAML(t *testing.T) {
	_, _, err := markdown.ParseFrontmatter[map[string]any](strings.NewReader("---\n: : :\n---\n"))
	assert.Error(t, err)
}
