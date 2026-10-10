# must

Unwrap results or panic when an error occurs.

## Installation

```sh
go get github.com/alextanhongpin/core/types/must
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/must"
	"strconv"
)

func main() {
	value := must.Value(strconv.Atoi("42"))
	fmt.Println(value)
}
```

## Behavior and limits

Value returns the value when err is nil and panics otherwise. NoError panics on a non-nil error. Use for startup wiring or inputs whose validity is already established; handle expected runtime failures normally.
