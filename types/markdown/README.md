# markdown

Read and write Markdown with typed YAML frontmatter.

## Installation

```sh
go get github.com/alextanhongpin/core/types/markdown
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/markdown"
	"io"
	"strings"
)

func main() {
	meta, body, err := markdown.ParseFrontmatter[map[string]any](strings.NewReader("---\ntitle: Hello\n---\nContent"))
	if err != nil {
		panic(err)
	}
	content, err := io.ReadAll(body)
	if err != nil {
		panic(err)
	}
	fmt.Println(meta, string(content))
}
```

## Behavior and limits

WriteFrontmatter writes YAML between --- delimiters; Write and WriteString append content. Loader wraps an object implementing io.ReaderFrom and io.WriterTo, persists created_at metadata, and refreshes based on a TTL. Call the function returned by Sync to stop its background routine. Load and Save return filesystem and serialization errors.
