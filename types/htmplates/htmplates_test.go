package htmplates_test

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"testing"

	"github.com/alextanhongpin/core/types/htmplates"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/*.html
var templates embed.FS

func TestRender(t *testing.T) {
	t.Run("no hot-reload", func(t *testing.T) {
		engine := htmplates.Engine{
			BaseDir: "testdata",
			FS:      templates,
			Funcs: template.FuncMap{
				"Greet": func(msg string) string {
					return fmt.Sprintf("Hello, %s!", msg)
				},
			},
		}

		tpl := engine.Compile("function.html")()

		var b bytes.Buffer
		err := tpl.Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, world!\n", b.String())
	})

	t.Run("subtemplate", func(t *testing.T) {
		engine := htmplates.Engine{
			BaseDir: "testdata",
			FS:      templates,
		}

		tpl := engine.Compile("index.html", "extra.html")()

		var b bytes.Buffer
		err := tpl.Execute(&b, nil)

		is := assert.New(t)
		is.Nil(err)
		is.Equal("Hello, \nworld\n\n", b.String())
	})
}
