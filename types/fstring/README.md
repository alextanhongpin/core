# fstring

Format typed values using Go text templates.

## Installation

```sh
go get github.com/alextanhongpin/core/types/fstring
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/fstring"
)

func main() {
	f := fstring.FString[string]("Hello {{ . }}")
	text, err := f.Format("world")
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
```

## Behavior and limits

FormatFunc accepts a template.FuncMap; the provided FuncMap contains json and parse_json. FStringFromFile reads a template from disk. Invalid template syntax panics because parsing uses template.Must; execution errors are returned. Templates use text/template and do not escape HTML.
