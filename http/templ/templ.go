package templ

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"path"
	"slices"
)

// Template parses a set of files from FS with html/template, so values are
// escaped for their HTML, JavaScript, CSS or URL context.
//
// The first pattern names the entry template: Execute runs the template
// named after its base name (for example "base.html"), so it must be a file
// name, not a glob.
type Template struct {
	cached   *template.Template
	Patterns []string

	// FS is the filesystem to load templates from (e.g. os.DirFS(".") or embed.FS)
	FS fs.FS

	// Funcs provides custom functions available in templates
	Funcs template.FuncMap

	// HotReload re-parses templates on each render (for development). Parse
	// errors are then returned by Execute instead of panicking.
	HotReload bool
}

// Compile adds patterns and parses the templates. It panics if parsing
// fails, so call it during startup; it returns t for chaining.
func (t *Template) Compile(patterns ...string) *Template {
	t.Patterns = append(t.Patterns, patterns...)

	tpl, err := t.compile()
	if err != nil {
		panic(err)
	}
	t.cached = tpl

	return t
}

// Execute renders the entry template.
func (t *Template) Execute(wr io.Writer, data any) error {
	tpl, err := t.template()
	if err != nil {
		return err
	}

	return tpl.Execute(wr, data)
}

// ExecuteTemplate renders the named template.
func (t *Template) ExecuteTemplate(wr io.Writer, name string, data any) error {
	tpl, err := t.template()
	if err != nil {
		return err
	}

	return tpl.ExecuteTemplate(wr, name, data)
}

func (t *Template) template() (*template.Template, error) {
	if t.HotReload {
		return t.compile()
	}
	if t.cached == nil {
		return nil, fmt.Errorf("templ: template not compiled")
	}

	return t.cached, nil
}

func (t *Template) compile() (*template.Template, error) {
	if len(t.Patterns) == 0 {
		return nil, fmt.Errorf("templ: no patterns")
	}

	tpl, err := template.New("").Funcs(t.Funcs).ParseFS(t.FS, t.Patterns...)
	if err != nil {
		return nil, err
	}

	// ParseFS names each template after its file's base name; the entry
	// point is the first pattern.
	name := path.Base(t.Patterns[0])
	entry := tpl.Lookup(name)
	if entry == nil {
		return nil, fmt.Errorf("templ: entry template %q not found; the first pattern must be a file name", name)
	}

	return entry, nil
}

// Clone returns an uncompiled copy with the same patterns, so more files can
// be added with Compile without affecting t.
func (t *Template) Clone() *Template {
	return &Template{
		FS:        t.FS,
		Funcs:     maps.Clone(t.Funcs),
		HotReload: t.HotReload,
		Patterns:  slices.Clone(t.Patterns),
	}
}
