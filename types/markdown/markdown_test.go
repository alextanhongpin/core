package markdown_test

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/markdown"
	"github.com/go-openapi/testify/assert"
)

type mockLoader struct {
	val  string
	i    int
	vals []string
}

func (m *mockLoader) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprint(w, m.vals[m.i])
	m.i++
	return int64(n), err
}

func (m *mockLoader) ReadFrom(r io.Reader) (int64, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.val = string(b)

	return int64(len(b)), nil
}

func TestLoader(t *testing.T) {
	file := fmt.Sprintf("testdata/%s.md", t.Name())
	_ = os.RemoveAll(file)

	ml := &mockLoader{vals: []string{"foo", "bar"}}
	loader := markdown.NewLoader(
		file,
		map[string]any{
			"foo": "bar",
		},
		1*time.Second,
		ml,
	)

	err := loader.Load()
	is := assert.New(t)
	is.NoError(err)

	is.Equal("foo", ml.val)
	time.Sleep(time.Second)

	err = loader.Load()
	is.NoError(err)
	is.Equal("bar", ml.val)
}
