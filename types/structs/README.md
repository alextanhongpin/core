# structs

Inspect values and zero exported struct fields through reflection.

## Installation

```sh
go get github.com/alextanhongpin/core/types/structs
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/structs"
)

func main() {
	v := struct {
		Name string
		Age  int
	}{"Alice", 42}
	if err := structs.ZeroField(&v, "Age"); err != nil {
		panic(err)
	}
	fmt.Println(v)
}
```

## Behavior and limits

ZeroField recursively clears matching field names. ZeroPath targets dot-separated nested fields. Both accept struct pointers and slices of structs or struct pointers; writable fields must be exported. Nil pointers are skipped. GetMethodNames lists exported value and pointer methods; IsNilOrZero recognizes typed nil values and Go zero values. Reflection mutations can leave partial changes on error.
