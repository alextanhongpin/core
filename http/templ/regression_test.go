package templ_test

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/alextanhongpin/core/http/templ"
)

func TestExecuteEscapesHTML(t *testing.T) {
	// Regression: templates were parsed with text/template, so data was
	// written unescaped (XSS).
	tpl := (&templ.Template{FS: fstest.MapFS{
		"page.html": {Data: []byte(`<p>{{ . }}</p>`)},
	}}).Compile("page.html")

	var b bytes.Buffer
	if err := tpl.Execute(&b, `<script>alert(1)</script>`); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<script>") {
		t.Fatalf("output not escaped: %s", b.String())
	}
}

func TestHotReloadReturnsParseErrors(t *testing.T) {
	// Regression: a template edited into an invalid state panicked the
	// server on the next render.
	fs := newFS()
	tpl := (&templ.Template{FS: fs, HotReload: true}).Compile("base.html", "home.html")

	fs["home.html"] = &fstest.MapFile{Data: []byte(`{{ define "content" }}{{ end`)}
	if err := tpl.Execute(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("Execute = nil, want parse error")
	}
}

func TestGlobFirstPatternIsAnError(t *testing.T) {
	// Regression: a glob as the first pattern looked up a template named
	// "*.html", got nil, and Execute panicked with a nil dereference.
	tpl := &templ.Template{FS: newFS(), HotReload: true, Patterns: []string{"*.html"}}
	if err := tpl.Execute(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("Execute = nil, want error")
	}

	if err := (&templ.Template{FS: newFS()}).Execute(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("Execute on an uncompiled template = nil, want error")
	}
}
