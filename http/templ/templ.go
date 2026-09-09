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
	Patterns []string

	// FS is the filesystem to load templates from (e.g. os.DirFS(".") or embed.FS)
	FS fs.FS

	// Funcs provides custom functions available in templates
	Funcs template.FuncMap

	// HotReload reloads templates on each render (for development)
	// Only works with os.DirFS, not embed.FS.
	HotReload bool
}

func (t *Template) Compile(Patterns ...string) *Template {
	t.Patterns = append(t.Patterns, Patterns...)
	// ParseFS returns the first file, which is the "" in the template.New("").
	// We want to lookup the first file we passed in instead.
	t.cached = t.compile()

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
	return template.Must(template.New("").Funcs(t.Funcs).ParseFS(t.FS, t.Patterns...)).Lookup(filepath.Base(t.Patterns[0]))
}

func (t *Template) Clone() *Template {
	return &Template{
		FS:        t.FS,
		Funcs:     maps.Clone(t.Funcs),
		HotReload: t.HotReload,
		Patterns:  slices.Clone(t.Patterns),
	}
}
