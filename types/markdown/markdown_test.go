package markdown_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/markdown"
	"github.com/go-openapi/testify/assert"
)

// generator produces vals in turn and records the body read back.
type generator struct {
	vals  []string
	i     int
	got   string
	reads int
}

func (g *generator) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, g.vals[g.i%len(g.vals)])
	g.i++
	return int64(n), err
}

func (g *generator) ReadFrom(r io.Reader) (int64, error) {
	b, err := io.ReadAll(r)
	g.got = string(b)
	g.reads++
	return int64(len(b)), err
}

func TestLoader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dir", "cache.md")
	g := &generator{vals: []string{"foo", "bar"}}
	loader := markdown.NewLoader(path, map[string]any{"foo": "bar"}, 50*time.Millisecond, g)

	is := assert.New(t)
	is.NoError(loader.Load())
	is.Equal("foo", g.got)

	b, err := os.ReadFile(path)
	is.NoError(err)
	is.Contains(string(b), "foo: bar\n")
	is.Contains(string(b), "created_at: ")

	// Not expired: the cached file is read back.
	is.NoError(loader.Load())
	is.Equal("foo", g.got)
	is.Equal(1, g.i)

	time.Sleep(60 * time.Millisecond)
	is.NoError(loader.Load())
	is.Equal("bar", g.got)
	is.Equal(2, g.i)
}

// Regression: a nil meta map used to panic in Save.
func TestLoaderNilMeta(t *testing.T) {
	g := &generator{vals: []string{"x"}}
	loader := markdown.NewLoader(filepath.Join(t.TempDir(), "a.md"), nil, time.Hour, g)
	assert.NoError(t, loader.Load())
	assert.Equal(t, "x", g.got)
}

// Regression: a TTL shorter than the stored timestamp's precision used to
// recurse Save/Load forever.
func TestLoaderZeroTTL(t *testing.T) {
	g := &generator{vals: []string{"a", "b", "c"}}
	loader := markdown.NewLoader(filepath.Join(t.TempDir(), "a.md"), nil, 0, g)
	assert.NoError(t, loader.Load())
	assert.NoError(t, loader.Load())
	assert.Equal(t, "b", g.got)
	assert.Equal(t, 2, g.i)
}

// Regression: the stop function used to leak the goroutine.
func TestLoaderSync(t *testing.T) {
	g := &generator{vals: []string{"a"}}
	loader := markdown.NewLoader(filepath.Join(t.TempDir(), "a.md"), nil, 5*time.Millisecond, g)
	stop := loader.Sync()
	time.Sleep(30 * time.Millisecond)
	stop()
	stop()
	assert.NoError(t, loader.Load()) // no data race with the stopped goroutine
	assert.Equal(t, "a", g.got)
}

func ExampleParseFrontmatter() {
	src := "---\ntitle: Hello\n---\nContent"
	meta, body, err := markdown.ParseFrontmatter[map[string]any](strings.NewReader(src))
	if err != nil {
		panic(err)
	}
	content, _ := io.ReadAll(body)
	fmt.Println(meta["title"], string(content))
	// Output: Hello Content
}

func ExampleWriteString() {
	_ = markdown.WriteString(os.Stdout, map[string]any{"title": "Hello"}, "Content\n")
	// Output:
	// ---
	// title: Hello
	// ---
	// Content
}
