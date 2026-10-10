# multicloser

Close several resources and collect their errors.

## Installation

```sh
go get github.com/alextanhongpin/core/types/multicloser
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"github.com/alextanhongpin/core/types/multicloser"
	"io"
	"strings"
)

func main() {
	closers := multicloser.MultiCloser{io.NopCloser(strings.NewReader("data"))}
	if err := closers.Close(); err != nil {
		panic(err)
	}
}
```

## Behavior and limits

Close calls closers in slice order and returns errors.Join of all errors. Every closer is attempted even if an earlier one fails. Nil closers are skipped; repeated Close calls repeat every underlying call.
