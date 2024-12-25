package templ

import (
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"
	"text/template"
)

type Template struct {
	BasePath string
	// e.g. os.DirFS(".") or from embed.FS
	FS        fs.FS
	Funcs     template.FuncMap
	HotReload bool
}

func (t *Template) Compile(files ...string) *Extension {
	return &Extension{
		fn: t.ParseFunc(files...),
		t:  t,
	}
}

func (t *Template) ParseFunc(files ...string) func() *template.Template {
	if t.HotReload {
		return func() *template.Template {
			return t.Parse(files...)
		}
	}

	// Compile immediately, helps to check errors during runtime.
	tpl := t.Parse(files...)
	return func() *template.Template {
		return tpl
	}
}

func (t *Template) Parse(files ...string) *template.Template {
	f := t.paths(files...)

	tpl := template.Must(template.New("").Funcs(t.Funcs).ParseFS(t.FS, f...))
	// ParseFS returns the first file, which is the "" in the template.New("").
	// We want to lookup the first file we passed in instead.
	return tpl.Lookup(filepath.Base(f[0]))
}

func (t *Template) paths(files ...string) []string {
	if t.BasePath == "" {
		return files
	}

	f := slices.Clone(files)
	for i := range f {
		f[i] = filepath.Join(t.BasePath, f[i])
	}
	return f
}

type Extension struct {
	fn func() *template.Template
	t  *Template
}

func (e *Extension) Extend(files ...string) *Extension {
	return &Extension{
		fn: e.templateFunc(files...),
		t:  e.t,
	}
}

func (e *Extension) Execute(wr io.Writer, data any) error {
	return e.fn().Execute(wr, data)
}

func (e *Extension) ExecuteTemplate(wr io.Writer, name string, data any) error {
	return e.fn().ExecuteTemplate(wr, name, data)
}

func (e *Extension) Template() *template.Template {
	return e.fn()
}

func (e *Extension) templateFunc(files ...string) func() *template.Template {
	fn := func() *template.Template {
		top := template.Must(e.fn().Clone())
		return template.Must(top.ParseFS(e.t.FS, e.t.paths(files...)...))
	}
	if e.t.HotReload {
		return fn
	}

	return sync.OnceValue(fn)
}
