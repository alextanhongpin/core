# jsonb

Traverse and mutate decoded JSON maps and slices using path patterns.

## Installation

```sh
go get github.com/alextanhongpin/core/types/jsonb
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/jsonb"
)

func main() {
	data := map[string]any{"name": "Alice"}
	if err := jsonb.Set(data, "name", "Bob"); err != nil {
		panic(err)
	}
	values, err := jsonb.Get(data, "name")
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
}
```

## Behavior and limits

Use map[string]any and []any, as produced by decoding into any. Dot paths support * for one segment; arrays are visited through paths such as items[] and items[0]. Set replaces matching existing values only when their Go types match, otherwise returning ErrType. Delete removes matching map keys. Reviver traverses children before parents and mutates in place; errors can leave partial changes. Returning ErrSkip stops traversal and is treated as success.
