# markdown

Read and write Markdown files with YAML frontmatter, and cache generated
content in such a file with a time-to-live.

**Use it for:** frontmatter in notes, prompts, or static content, and for
caching an expensive generated document on disk.

**Don't use it for:** rendering Markdown to HTML, TOML or JSON frontmatter,
or caches shared between processes.

## Installation

```sh
go get github.com/alextanhongpin/core/types/markdown
```

The package is part of the `github.com/alextanhongpin/core/types` module;
check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alextanhongpin/core/types/markdown"
)

func main() {
	_ = markdown.WriteString(os.Stdout, map[string]any{"title": "Hello"}, "Content\n")

	src := "---\ntitle: Hello\n---\nContent"
	meta, body, err := markdown.ParseFrontmatter[map[string]any](strings.NewReader(src))
	if err != nil {
		panic(err)
	}
	content, _ := io.ReadAll(body)
	fmt.Println(meta["title"], string(content))
}
```

It prints:

```
---
title: Hello
---
Content
Hello Content
```

## API

```go
var ErrUnterminated error

func WriteFrontmatter(w io.Writer, meta any) error
func Write(w io.Writer, meta any, content []byte) error
func WriteString(w io.Writer, meta any, content string) error
func ParseFrontmatter[T any](r io.Reader) (T, io.Reader, error)

type Content interface {
	io.WriterTo   // generates the body
	io.ReaderFrom // receives the body read from disk
}

func NewLoader(path string, meta map[string]any, ttl time.Duration, content Content) *Loader
func (l *Loader) Load() error
func (l *Loader) Save() error
func (l *Loader) Sync() (stop func())
```

## Behavior and guarantees

- Frontmatter is YAML (via `sigs.k8s.io/yaml`, so struct fields use `json`
  tags) between a first line of `---` and the next `---` line. `\r\n` line
  endings are accepted.
- Without frontmatter, `ParseFrontmatter` returns the zero `T` and a reader
  over the whole input, including for empty or short input. An opening
  delimiter without a closing one returns `ErrUnterminated`; empty
  frontmatter returns the zero `T`.
- `Loader.Load` regenerates the file with `Content.WriteTo` when it is
  missing or its `created_at` is at least `ttl` old, then passes the body to
  `Content.ReadFrom`. A nonpositive `ttl` regenerates on every load; a file
  without a parsable `created_at` never expires.
- `Save` writes `meta` plus `created_at` (UTC, RFC 3339 with nanoseconds),
  creating parent directories. It truncates the file before writing, so a
  failed write leaves a partial file.
- `Sync` calls `Load` every `ttl` and logs errors with `slog`; it panics if
  `ttl` is not positive. `stop` stops the goroutine, waits for it, and may be
  called more than once.
- A `Loader` serializes its own calls, but does not lock the file against
  other processes.

## Testing

```sh
go vet ./markdown
go test -race -count=1 ./markdown
```
