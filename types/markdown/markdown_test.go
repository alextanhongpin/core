package markdown_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/markdown"
	"github.com/go-openapi/testify/assert"
)

type mockLoader struct {
}

func (m *mockLoader) Write(w io.Writer) error {
	_, err := fmt.Fprint(w, "hello")
	return err
}

func (m *mockLoader) Read(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

func TestLoader(t *testing.T) {
	loader := markdown.NewLoader(
		"testdata/"+t.Name()+".md",
		map[string]any{
			"foo": "bar",
		},
		10*time.Second,
		&mockLoader{},
	)

	err := loader.Sync()
	is := assert.New(t)
	is.NoError(err)

	res := loader.Load()
	t.Log(res)
}
