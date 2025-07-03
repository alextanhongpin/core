# HTTP Template Package

The HTTP Template package provides a flexible, hot-reloadable HTML template system for Go web applications with support for layout composition and embedded files.

## Features

- **Hot-Reload**: Automatic template reloading during development
- **Layout Composition**: Support for base layouts and template extension
- **Embedded Files**: Compatible with Go 1.16+ `embed.FS`
- **Template Functions**: Custom function maps for template rendering
- **Caching**: Production-ready template caching
- **Error Handling**: Robust error handling for template issues

## Quick Start

```go
package main

import (
    "net/http"
    "os"
    
    "github.com/alextanhongpin/core/http/templ"
)

func main() {
    // Create template instance with hot-reload in development
    tpl := &templ.Template{
        FS:        os.DirFS("templates"),
        HotReload: true, // Set to false in production
    }
    
    // Compile templates
    homePage := tpl.Compile("base.html", "pages/home.html")
    aboutPage := tpl.Compile("base.html", "pages/about.html")
    
    // HTTP handlers
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

#### `Template` struct

Main template type with configuration options:

```go
type Template struct {
    // FS is the filesystem to load templates from (os.DirFS or embed.FS)
    FS fs.FS
    
    // HotReload reloads templates on each render (for development)
    HotReload bool
    
    // FuncMap provides functions to the templates
    FuncMap template.FuncMap
}
```

### Template Compilation

#### `(t *Template) Compile(filenames ...string) *CompiledTemplate`

Compiles one or more templates into a single template.

```go
// Simple template
simple := tpl.Compile("page.html")

// Layout with content
home := tpl.Compile("base.html", "pages/home.html")

// Multiple partials
dashboard := tpl.Compile("base.html", "partials/*.html", "pages/dashboard.html")
```

### Template Rendering

#### `(ct *CompiledTemplate) Execute(w io.Writer, data any) error`

Renders a compiled template with provided data.

```go
data := map[string]any{
    "Title": "Welcome",
    "Items": []string{"Apple", "Banana", "Cherry"},
}

err := homePage.Execute(w, data)
```

### Template Extension

#### `(ct *CompiledTemplate) Extend(filenames ...string) *CompiledTemplate`

Creates a new template that extends an existing compiled template.

```go
// Compile base layout with common partials
base := tpl.Compile("base.html", "partials/*.html")

// Extend base with specific page content
home := base.Extend("pages/home.html")
about := base.Extend("pages/about.html")
```

## Using with Embedded Files

Go 1.16+ allows embedding files directly into your binary:

```go
package main

import (
    "embed"
    "net/http"
    
    "github.com/alextanhongpin/core/http/templ"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
    tpl := &templ.Template{
        FS:        templateFS,
        HotReload: false, // No need for hot-reload with embedded files
    }
    
    homePage := tpl.Compile("templates/base.html", "templates/pages/home.html")
    
    // HTTP handler
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        homePage.Execute(w, map[string]any{
            "Title": "Home Page",
        })
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Custom Function Maps

Provide custom functions to your templates:

```go
tpl := &templ.Template{
    FS: os.DirFS("templates"),
    FuncMap: template.FuncMap{
        "formatDate": func(t time.Time) string {
            return t.Format("Jan 02, 2006")
        },
        "capitalize": strings.ToUpper,
        "add": func(a, b int) int {
            return a + b
        },
    },
}
```

## Layout Composition Pattern

The template package makes layout composition easy:

### Base Layout (`base.html`)

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Default Title{{end}}</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <header>
        {{block "header" .}}Default Header{{end}}
    </header>
    
    <main>
        {{block "content" .}}Default Content{{end}}
    </main>
    
    <footer>
        {{block "footer" .}}Default Footer{{end}}
    </footer>
</body>
</html>
```

### Content Page (`pages/home.html`)

```html
{{define "title"}}Home Page{{end}}

{{define "header"}}
    <h1>Welcome to our site</h1>
    <nav>
        <a href="/">Home</a>
        <a href="/about">About</a>
    </nav>
{{end}}

{{define "content"}}
    <h2>Welcome, {{.User}}!</h2>
    <p>This is the home page content.</p>
{{end}}
```

## Development vs. Production

Configure the template system differently for development and production:

```go
tpl := &templ.Template{
    FS:        templateFS,
    HotReload: os.Getenv("ENV") != "production",
}
```

## Error Handling

The template package provides detailed error information:

```go
homePage := tpl.Compile("base.html", "pages/home.html")

if err := homePage.Execute(w, data); err != nil {
    // Log the error
    log.Printf("Template error: %v", err)
    
    // Send an error response
    http.Error(w, "Template rendering failed", http.StatusInternalServerError)
    return
}
```

## Best Practices

1. **Directory Structure**: Organize templates into logical directories (layouts, pages, partials)
2. **Disable Hot-Reload in Production**: Set `HotReload: false` in production for better performance
3. **Error Handling**: Always check template execution errors
4. **Use Embedded Files in Production**: Embed templates in your binary for production deployments
5. **Template Composition**: Use template inheritance patterns for consistent layouts
