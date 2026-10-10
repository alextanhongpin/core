# useragent

Fetch browser user-agent strings or load them from a text stream.

## Installation

```sh
go get github.com/alextanhongpin/core/types/useragent
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/useragent"
	"strings"
)

func main() {
	loader := useragent.NewLoader([]string{useragent.Firefox})
	if _, err := loader.ReadFrom(strings.NewReader("ExampleAgent/1.0\n")); err != nil {
		panic(err)
	}
	fmt.Println(loader.Load())
}
```

## Behavior and limits

For fetches strings from useragentstring.com using the default HTTP client. WriteTo fetches configured browsers and writes one string per line; it requires network access. ReadFrom loads newline-separated strings locally. Call ReadFrom successfully before Load, otherwise Load dereferences a nil pointer. Load returns a cloned slice. Fetch results depend on the external site and its HTML structure.
