package htmplates_test

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"testing"

	"github.com/alextanhongpin/core/types/htmplates"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/*.html testdata/**/*.html
var templates embed.FS

func TestRender(t *testing.T) {
	t.Run("local dir", func(t *testing.T) {
		html5 := htmplates.Engine{
			BaseDir: "testdata",
			FS:      os.DirFS("."),
			Funcs: template.FuncMap{
				"Greet": func(msg string) string {
					return fmt.Sprintf("Hello, %s!", msg)
				},
			},
		}

		tf := html5.Compile("function.html")

		var b bytes.Buffer
		err := tf().Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, world!\n", b.String())
	})

	t.Run("no hot-reload", func(t *testing.T) {
		html5 := htmplates.Engine{
			BaseDir: "testdata",
			FS:      templates,
			Funcs: template.FuncMap{
				"Greet": func(msg string) string {
					return fmt.Sprintf("Hello, %s!", msg)
				},
			},
		}

		tf := html5.Compile("function.html")

		var b bytes.Buffer
		err := tf().Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, world!\n", b.String())

		html5.Funcs = template.FuncMap{
			"Greet": func(msg string) string {
				return "hello world"
			},
		}

		b.Reset()
		err = tf().Execute(&b, nil)
		is.Nil(err)
		is.Equal("Hello, world!\n", b.String())
	})

	t.Run("hot-reload", func(t *testing.T) {
		html5 := htmplates.Engine{
			BaseDir:   "testdata",
			FS:        templates,
			HotReload: true,
			Funcs: template.FuncMap{
				"Greet": func(msg string) string {
					return fmt.Sprintf("Hello, %s!", msg)
				},
			},
		}

		tf := html5.Compile("function.html")

		var b bytes.Buffer
		err := tf().Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, world!\n", b.String())

		html5.Funcs = template.FuncMap{
			"Greet": func(msg string) string {
				return "hello world"
			},
		}

		b.Reset()
		err = tf().Execute(&b, nil)
		is.Nil(err)
		is.Equal("hello world\n", b.String())
	})

	t.Run("subtemplate", func(t *testing.T) {
		html5 := htmplates.Engine{
			BaseDir: "testdata",
			FS:      templates,
		}

		tpl := html5.Compile("index.html", "extra.html")()

		var b bytes.Buffer
		err := tpl.Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, \nworld\n\n", b.String())
	})

	t.Run("subtemplates", func(t *testing.T) {
		html5 := htmplates.Engine{
			BaseDir: "testdata",
			FS:      templates,
		}

		tpl := html5.Compile("base.html", "components/*.html")()

		var b bytes.Buffer
		err := tpl.Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("\nheader\n\n\nfooter\n\n", b.String())
	})
}
