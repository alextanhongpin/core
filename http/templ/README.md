# templ

Load HTML templates from an `fs.FS`, compose layouts and partials, and
optionally re-parse on every render during development. Templates use
`html/template`, so data is escaped for its context.

```go
base := (&templ.Template{
	FS:        os.DirFS("templates"), // or an embed.FS (fs.Sub it to the template root)
	HotReload: os.Getenv("ENV") == "dev",
	Funcs:     template.FuncMap{"upper": strings.ToUpper},
}).Compile("base.html", "partials/*.html")

home := base.Clone().Compile("pages/home.html")
about := base.Clone().Compile("pages/about.html")

mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
	if err := home.Execute(w, map[string]any{"Title": "Home"}); err != nil {
		http.Error(w, "render failed", http.StatusInternalServerError)
	}
})
```

- `Compile(patterns...)` appends patterns, parses them, and panics on error
  (call it at startup). The first pattern must be a file name: `Execute`
  renders the template named after its base name.
- `Execute(w, data)` / `ExecuteTemplate(w, name, data)` render. With
  `HotReload` they re-parse first and return parse errors instead of
  panicking.
- `Clone()` copies the patterns and functions without the compiled state, so
  pages can extend a shared layout.

A `Template` must not be compiled while it is being executed concurrently.
Execution writes directly to `w`: render into a buffer first if a failed
render must not leave a partial response.
