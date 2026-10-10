# tuples

Hold two typed values in a single generic struct.

## Installation

```sh
go get github.com/alextanhongpin/core/types/tuples
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/tuples"
)

func main() {
	pair := tuples.Tuple[string, int]{T1: "count", T2: 42}
	fmt.Println(pair.T1, pair.T2)
}
```

## Behavior and limits

Tuple exposes T1 and T2 directly. Its zero value contains the zero values of both type parameters. It defines no ordering or serialization behavior beyond ordinary Go struct behavior.
