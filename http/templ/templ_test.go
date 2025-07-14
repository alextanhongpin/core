package templ_test

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/alextanhongpin/core/http/templ"
)

func TestCompile(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	var b bytes.Buffer
	err := tpl.Compile("base.html", "*.html").Execute(&b, nil)
	if err != nil {
		t.Fatalf("Compile = %v, want nil", err)
	}
	if want, got := "home", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtend(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	base := tpl.Compile("base.html", "partials/*.html")
	home := base.Extend("home.html")
	about := base.Extend("about.html")

	var b bytes.Buffer
	err := home.Execute(&b, nil)
	if err != nil {
		t.Fatalf("Execute = %v, want nil", err)
	}
	if want, got := "home", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	b.Reset()
	err = about.Execute(&b, nil)
	if err != nil {
		t.Fatalf("Execute = %v, want nil", err)
	}
	if want, got := "about", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestPartial(t *testing.T) {
	tpl := &templ.Template{
		FS: newFS(),
	}
	page := tpl.Compile("page.html", "partials/*.html")

	var b bytes.Buffer
	err := page.Execute(&b, nil)
	if err != nil {
		t.Fatalf("Execute = %v, want nil", err)
	}

	if want, got := "header page footer", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	b.Reset()
	err = page.ExecuteTemplate(&b, "header", nil)
	if err != nil {
		t.Fatalf("ExecuteTemplate = %v, want nil", err)
	}
	if want, got := "header", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	b.Reset()
	err = page.ExecuteTemplate(&b, "footer", nil)
	if err != nil {
		t.Fatalf("ExecuteTemplate = %v, want nil", err)
	}
	if want, got := "footer", b.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestHotReload(t *testing.T) {
	t.Run("no hot-reload", func(t *testing.T) {
		fs := newFS()
		tpl := &templ.Template{
			FS:        fs,
			HotReload: false,
		}

		var b bytes.Buffer
		home := tpl.Compile("base.html", "home.html")
		err := home.Execute(&b, map[string]any{
			"Msg": "world",
		})
		if err != nil {
			t.Fatalf("Execute = %v, want nil", err)
		}
		if want, got := "home", b.String(); want != got {
			t.Errorf("want %q, got %q", want, got)
		}

		fs["home.html"] = &fstest.MapFile{
			Data: []byte(`{{ define "content" }}hi, {{.Msg}}{{ end }}`),
		}

		b.Reset()
		err = home.Execute(&b, map[string]any{
			"Msg": "world",
		})
		if err != nil {
			t.Fatalf("Execute = %v, want nil", err)
		}
		if want, got := "home", b.String(); want != got {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("hot-reload", func(t *testing.T) {
		fs := newFS()
		tpl := &templ.Template{
			FS:        fs,
			HotReload: true,
		}

		var b bytes.Buffer
		home := tpl.Compile("base.html", "home.html")
		err := home.Execute(&b, map[string]any{
			"Msg": "world",
		})
		if err != nil {
			t.Fatalf("Execute = %v, want nil", err)
		}
		if want, got := "home", b.String(); want != got {
			t.Errorf("want %q, got %q", want, got)
		}

		fs["home.html"] = &fstest.MapFile{
			Data: []byte(`{{ define "content" }}hi, {{.Msg}}{{ end }}`),
		}

		b.Reset()
		err = home.Execute(&b, map[string]any{
			"Msg": "world",
		})
		if err != nil {
			t.Fatalf("Execute = %v, want nil", err)
		}
		if want, got := "hi, world", b.String(); want != got {
			t.Errorf("want %q, got %q", want, got)
		}
	})
}

func newFS() fstest.MapFS {
	return fstest.MapFS{
		"base.html": {
			Data: []byte(`{{ template "content" . }}`),
		},
		"home.html": {
			Data: []byte(`{{ define "content" }}home{{ end }}`),
		},
		"about.html": {
			Data: []byte(`{{ define "content" }}about{{ end }}`),
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
