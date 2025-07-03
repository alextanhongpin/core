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
	base := tpl.Compile("base.html", "partials/*.html")
	home := base.Extend("home.html")
	about := base.Extend("about.html")

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

func TestParseSafe(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}

	t.Run("valid templates", func(t *testing.T) {
		tmpl, err := tpl.ParseSafe("base.html")
		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
	})

	t.Run("no files provided", func(t *testing.T) {
		_, err := tpl.ParseSafe()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no template files provided")
	})

	t.Run("empty filename", func(t *testing.T) {
		_, err := tpl.ParseSafe("base.html", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty filename provided")
	})

	t.Run("template not found", func(t *testing.T) {
		_, err := tpl.ParseSafe("nonexistent.html")
		assert.Error(t, err)
	})
}

func TestCompileSafe(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}

	t.Run("valid templates", func(t *testing.T) {
		ext, err := tpl.CompileSafe("base.html", "home.html")
		assert.NoError(t, err)
		assert.NotNil(t, ext)

		var b bytes.Buffer
		err = ext.Execute(&b, map[string]any{"Msg": "world"})
		assert.NoError(t, err)
		assert.Equal(t, "hello, world", b.String())
	})

	t.Run("invalid templates in production mode", func(t *testing.T) {
		tpl.HotReload = false
		_, err := tpl.CompileSafe("nonexistent.html")
		assert.Error(t, err)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		tpl := &templ.Template{
			FS: newFS(),
		}
		assert.NoError(t, tpl.Validate())
	})

	t.Run("missing filesystem", func(t *testing.T) {
		tpl := &templ.Template{}
		err := tpl.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "filesystem (FS) is required")
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
