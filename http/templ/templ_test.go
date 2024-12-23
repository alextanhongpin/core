package templ_test

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/alextanhongpin/core/http/templ"
	"github.com/stretchr/testify/assert"
)

func TestCompile(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	var b bytes.Buffer
	err := tpl.Compile("base.html", "home.html").Execute(&b, map[string]any{
		"Msg": "world",
	})
	is := assert.New(t)
	is.Nil(err)
	is.Equal("hello, world", b.String())
}

func TestExtend(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	base := tpl.Compile("base.html")
	home := base.Extend("home.html")
	about := base.Extend("about.html", "partials/*.html")

	is := assert.New(t)
	var b bytes.Buffer
	is.Nil(home.Execute(&b, map[string]any{
		"Msg": "world",
	}))
	is.Equal("hello, world", b.String())

	b.Reset()
	is.Nil(about.Execute(&b, map[string]any{
		"Msg": "world",
	}))
	is.Equal("header: world", b.String())
}

func TestPartial(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	page := tpl.Compile("page.html", "partials/*.html")
	is := assert.New(t)
	var b bytes.Buffer
	is.Nil(page.Execute(&b, nil))
	is.Equal("header page footer", b.String())

	b.Reset()
	is.Nil(page.ExecuteTemplate(&b, "header", nil))
	is.Equal("header", b.String())

	b.Reset()
	is.Nil(page.ExecuteTemplate(&b, "footer", nil))
	is.Equal("footer", b.String())
}

func TestBasePath(t *testing.T) {
	fs := newFS()
	for k, v := range fs {
		fs["templates/"+k] = v
	}
	tpl := &templ.Template{
		FS:       fs,
		BasePath: "templates",
	}
	var b bytes.Buffer
	err := tpl.Compile("base.html", "home.html").Execute(&b, map[string]any{
		"Msg": "world",
	})
	is := assert.New(t)
	is.Nil(err)
	is.Equal("hello, world", b.String())
}

func TestHotReload(t *testing.T) {
	t.Run("no hot-reload", func(t *testing.T) {
		fs := newFS()
		tpl := &templ.Template{
			FS:        fs,
			HotReload: false,
		}
		var b bytes.Buffer
		homeTpl := tpl.Compile("base.html", "home.html")
		err := homeTpl.Execute(&b, map[string]any{
			"Msg": "world",
		})
		is := assert.New(t)
		is.Nil(err)
		is.Equal("hello, world", b.String())

		fs["home.html"] = &fstest.MapFile{
			Data: []byte(`{{ define "content" }}hi, {{.Msg}}{{ end }}`),
		}

		b.Reset()
		err = homeTpl.Execute(&b, map[string]any{
			"Msg": "world",
		})
		is.Nil(err)
		is.Equal("hello, world", b.String())
	})

	t.Run("hot-reload", func(t *testing.T) {
		fs := newFS()
		tpl := &templ.Template{
			FS:        fs,
			HotReload: true,
		}
		var b bytes.Buffer
		homeTpl := tpl.Compile("base.html", "home.html")
		err := homeTpl.Execute(&b, map[string]any{
			"Msg": "world",
		})
		is := assert.New(t)
		is.Nil(err)
		is.Equal("hello, world", b.String())

		fs["home.html"] = &fstest.MapFile{
			Data: []byte(`{{ define "content" }}hi, {{.Msg}}{{ end }}`),
		}

		b.Reset()
		err = homeTpl.Execute(&b, map[string]any{
			"Msg": "world",
		})
		is.Nil(err)
		is.Equal("hi, world", b.String())
	})
}

func newFS() fstest.MapFS {
	return fstest.MapFS{
		"base.html": {
			Data: []byte(`{{ template "content" . }}`),
		},
		"home.html": {
			Data: []byte(`{{ define "content" }}hello, {{.Msg}}{{ end }}`),
		},
		"about.html": {
			Data: []byte(`{{ define "content" }}{{ template "header" . }}: {{.Msg}}{{ end }}`),
		},
		"page.html": {
			Data: []byte(`{{ template "header" }} page {{ template "footer" }}`),
		},
		"partials/header.html": {
			Data: []byte(`{{ define "header" }}header{{ end }}`),
		},
		"partials/footer.html": {
			Data: []byte(`{{ define "footer" }}footer{{ end }}`),
		},
	}
}
