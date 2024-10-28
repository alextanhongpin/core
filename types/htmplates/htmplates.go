package htmplates

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"
)

type Engine struct {
	// e.g. os.DirFS(".") or from embed.FS
	FS        fs.FS
	Funcs     template.FuncMap
	BaseDir   string
	HotReload bool
}

func (e *Engine) Compile(files ...string) func() *template.Template {
	fn := func() *template.Template {
		return e.compile(slices.Clone(files)...)
	}
	if e.HotReload {
		return fn
	}

	return sync.OnceValue(fn)
}

func (e *Engine) compile(files ...string) *template.Template {
	if e.BaseDir != "" {
		for i := range files {
			files[i] = filepath.Join(e.BaseDir, files[i])
		}
	}
	// ParseFS will designate the template name as the first base path of the
	// first file.
	return template.Must(template.New("").Funcs(e.Funcs).ParseFS(e.FS, files...)).Lookup(filepath.Base(files[0]))
}
