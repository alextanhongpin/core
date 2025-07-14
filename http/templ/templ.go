package templ

import (
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"text/template"
)

type Template struct {
	cached   *template.Template
	patterns []string

	// FS is the filesystem to load templates from (e.g. os.DirFS(".") or embed.FS)
	FS fs.FS

	// Funcs provides custom functions available in templates
	Funcs template.FuncMap

	// HotReload reloads templates on each render (for development)
	// Only works with os.DirFS, not embed.FS.
	HotReload bool
}

func (t *Template) Compile(patterns ...string) *Template {
	t.patterns = slices.Clone(patterns)
	// ParseFS returns the first file, which is the "" in the template.New("").
	// We want to lookup the first file we passed in instead.

	if !t.HotReload {
		t.cached = t.compile()
	}

	return t
}

func (t *Template) Execute(wr io.Writer, data any) error {
	if t.HotReload {
		// If hot reload is enabled, compile the template each time
		return t.compile().Execute(wr, data)
	}

	return t.cached.Execute(wr, data)
}

func (t *Template) ExecuteTemplate(wr io.Writer, name string, data any) error {
	if t.HotReload {
		// If hot reload is enabled, compile the template each time
		return t.compile().ExecuteTemplate(wr, name, data)
	}

	return t.cached.ExecuteTemplate(wr, name, data)
}

func (t *Template) compile() *template.Template {
	return template.Must(template.New("").Funcs(t.Funcs).ParseFS(t.FS, t.patterns...)).Lookup(filepath.Base(t.patterns[0]))
}

func (t *Template) Extend(patterns ...string) *Template {
	tpl := &Template{
		FS:        t.FS,
		Funcs:     maps.Clone(t.Funcs),
		HotReload: t.HotReload,
	}

	return tpl.Compile(append(t.patterns, patterns...)...)
}
