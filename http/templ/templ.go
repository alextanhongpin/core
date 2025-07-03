package templ

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"
	"text/template"
)

// Template represents a template configuration with optional hot-reload support.
type Template struct {
	// BasePath is prepended to all template file paths
	BasePath string
	// FS is the filesystem to load templates from (e.g. os.DirFS(".") or embed.FS)
	FS fs.FS
	// Funcs provides custom functions available in templates
	Funcs template.FuncMap
	// HotReload reloads templates on each render (for development)
	HotReload bool
}

func (t *Template) Compile(files ...string) *Extension {
	return &Extension{
		fn: t.ParseFunc(files...),
		t:  t,
	}
}

// ParseSafe parses templates with error handling instead of panicking
func (t *Template) ParseSafe(files ...string) (*template.Template, error) {
	if err := validateFiles(files); err != nil {
		return nil, err
	}

	f := t.paths(files...)

	tpl, err := template.New("").Funcs(t.Funcs).ParseFS(t.FS, f...)
	if err != nil {
		return nil, err
	}

	// ParseFS returns the first file, which is the "" in the template.New("").
	// We want to lookup the first file we passed in instead.
	result := tpl.Lookup(filepath.Base(f[0]))
	if result == nil {
		return nil, fmt.Errorf("template %s not found", filepath.Base(f[0]))
	}

	return result, nil
}

// CompileSafe compiles templates with error handling
func (t *Template) CompileSafe(files ...string) (*Extension, error) {
	if !t.HotReload {
		// Validate templates at compile time in production
		if _, err := t.ParseSafe(files...); err != nil {
			return nil, err
		}
	}

	return &Extension{
		fn: t.ParseFunc(files...),
		t:  t,
	}, nil
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
	if err := t.Validate(); err != nil {
		panic(err)
	}

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

// validateFiles validates the input files before processing
func validateFiles(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no template files provided")
	}

	for _, file := range files {
		if file == "" {
			return fmt.Errorf("empty filename provided")
		}
	}

	return nil
}

// Validate validates the Template configuration
func (t *Template) Validate() error {
	if t.FS == nil {
		return fmt.Errorf("filesystem (FS) is required")
	}
	return nil
}

// Extension represents a compiled template that can be extended with additional files.
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
