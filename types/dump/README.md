# dump

Inspect Go values with reflection and pretty printing.

## Installation

```sh
go get github.com/alextanhongpin/core/types/dump
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/dump"
)

func main() {
	fmt.Println(dump.SDump(map[string]int{"count": 3}))
}
```

## Behavior and limits

Dump returns bytes and SDump returns a string. PrettyPrint writes to standard output; FPrettyPrint accepts an io.Writer and includes cycle detection. Values can customize their representation through the Dumper interface. Output is for diagnostics, not a stable serialization format.
