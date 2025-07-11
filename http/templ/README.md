# HTTP Template Package

The HTTP Template package provides a flexible, hot-reloadable HTML template system for Go web applications with support for layout composition and embedded files.

## Features

- **Hot-Reload**: Automatic template reloading during development
- **Layout Composition**: Support for base layouts and template extension
- **Embedded Files**: Compatible with Go 1.16+ `embed.FS`
- **Template Functions**: Custom function maps for template rendering
- **Caching**: Production-ready template caching
- **Error Handling**: Robust error handling for template issues
- **Cross-Package Integration**: Works with `handler`, `server`, and logging middleware

## Quick Start

```go
package main

import (
    "net/http"
    "os"
    "github.com/alextanhongpin/core/http/templ"
)

func main() {
    tpl := &templ.Template{
        FS:        os.DirFS("templates"),
        HotReload: true, // Set to false in production
    }
    homePage := tpl.Compile("base.html", "pages/home.html")
    aboutPage := tpl.Compile("base.html", "pages/about.html")
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data := map[string]any{
            "Title": "Home Page",
            "User":  "John Doe",
        }
        homePage.Execute(w, data)
    })
    http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
        data := map[string]any{
            "Title": "About Us",
        }
        aboutPage.Execute(w, data)
    })
    http.ListenAndServe(":8080", nil)
}
```

## API Reference

### Template Initialization

#### `Template{FS: fs.FS, HotReload: bool}`
Creates a new template instance.

### Compile

#### `Compile(layout string, files ...string) *CompiledTemplate`
Compiles templates with optional layout.

### Execute

#### `Execute(w http.ResponseWriter, data any) error`
Renders template to response writer.

## Best Practices

- Use hot-reload in development, caching in production.
- Organize templates with layouts and partials for maintainability.
- Integrate with `handler` for structured error handling.

## Related Packages

- [`handler`](../handler/README.md): Base handler utilities
- [`server`](../server/README.md): HTTP server utilities

## License

MIT
